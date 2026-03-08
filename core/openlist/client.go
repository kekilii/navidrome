package openlist

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/navidrome/navidrome/consts"
)

var httpClient = &http.Client{Timeout: consts.DefaultHttpClientTimeOut}

type tokenCache struct {
	mu    sync.Mutex
	token string
	exp   time.Time
}

var tokens tokenCache

type openListResp[T any] struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}

type openListLoginData struct {
	Token string `json:"token"`
}

type openListFSGetData struct {
	RawURL string `json:"raw_url"`
	IsDir  bool   `json:"is_dir"`
}

func ResolveRawURL(ctx context.Context, openListPath string) (string, error) {
	cfg := Current()
	if !IsConfigured(cfg) {
		return "", fmt.Errorf("openlist not configured")
	}
	openListPath = strings.TrimSpace(openListPath)
	if openListPath == "" {
		return "", fmt.Errorf("empty openlist path")
	}

	token, ok := tokens.get()
	if !ok {
		var err error
		token, err = login(ctx, cfg)
		if err != nil {
			return "", err
		}
	}

	rawURL, err := fsGet(ctx, openListPath, token, cfg)
	if err == nil {
		target := resolveRawURL(rawURL, cfg.OpenListBase)
		if target != "" {
			return target, nil
		}
	}

	token, err = login(ctx, cfg)
	if err != nil {
		return "", err
	}
	rawURL, err = fsGet(ctx, openListPath, token, cfg)
	if err != nil {
		return "", err
	}
	target := resolveRawURL(rawURL, cfg.OpenListBase)
	if target == "" {
		return "", fmt.Errorf("empty openlist raw url")
	}
	return target, nil
}

func login(ctx context.Context, cfg Config) (string, error) {
	payload := map[string]string{
		"username": cfg.OpenListUser,
		"password": cfg.OpenListPass,
	}
	b, _ := json.Marshal(payload)

	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, cfg.OpenListBase+"/api/auth/login", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var body openListResp[openListLoginData]
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", err
	}
	if body.Code != 200 {
		return "", fmt.Errorf("openlist login failed: code=%d message=%s", body.Code, body.Message)
	}
	token := strings.TrimSpace(body.Data.Token)
	if token != "" {
		tokens.set(token, 47*time.Hour)
	}
	return token, nil
}

func fsGet(ctx context.Context, openListPath, token string, cfg Config) (string, error) {
	payload := map[string]string{"path": openListPath}
	b, _ := json.Marshal(payload)

	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, cfg.OpenListBase+"/api/fs/get", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", token)

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var body openListResp[openListFSGetData]
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", err
	}
	if body.Code != 200 {
		return "", fmt.Errorf("openlist fs/get failed: code=%d message=%s", body.Code, body.Message)
	}
	if body.Data.IsDir {
		return "", fmt.Errorf("openlist fs/get returned directory for path: %s", openListPath)
	}
	return strings.TrimSpace(body.Data.RawURL), nil
}

func (c *tokenCache) get() (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.token == "" || time.Now().After(c.exp) {
		return "", false
	}
	return c.token, true
}

func (c *tokenCache) set(token string, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.token = token
	c.exp = time.Now().Add(ttl)
}

func (c *tokenCache) clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.token = ""
	c.exp = time.Time{}
}

func getenv(key string) string {
	return strings.TrimSpace(os.Getenv(key))
}

func getenvBool(key string, def bool) bool {
	return parseBool(strings.TrimSpace(os.Getenv(key)), def)
}

func parseBool(value string, def bool) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "y", "on":
		return true
	case "0", "false", "no", "n", "off":
		return false
	default:
		return def
	}
}

func boolToString(v bool) string {
	if v {
		return "true"
	}
	return "false"
}
