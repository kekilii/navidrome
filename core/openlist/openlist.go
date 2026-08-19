package openlist

import (
	"cmp"
	"context"
	"crypto/sha256"
	"errors"
	"strings"
	"sync"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils"
)

const (
	RecordID         = "openlist"
	EnabledKey       = "OpenListEnabled"
	BaseKey          = "OpenListBase"
	UserKey          = "OpenListUser"
	PassKey          = "OpenListPass"
	CoverEnabledKey  = "OpenListCoverEnabled"
	StreamEnabledKey = "OpenListStreamEnabled"
)
const defaultOpenListBase = "http://openlist:5244"

type Config struct {
	Enabled       bool   `json:"enabled"`
	OpenListBase  string `json:"openlistBase"`
	OpenListUser  string `json:"openlistUser"`
	OpenListPass  string `json:"openlistPass"`
	CoverEnabled  bool   `json:"coverEnabled"`
	StreamEnabled bool   `json:"streamEnabled"`
}

var (
	stateMu sync.RWMutex
	state   = Config{
		Enabled:       true,
		CoverEnabled:  true,
		StreamEnabled: true,
	}
)

func Bootstrap(ds model.DataStore) error {
	cfg := Config{
		Enabled:       getenvBool("OPENLIST_ENABLED", true),
		OpenListBase:  normalizeBase(cmp.Or(getenv("OPENLIST_BASE"), defaultOpenListBase)),
		OpenListUser:  strings.TrimSpace(getenv("OPENLIST_USER")),
		OpenListPass:  strings.TrimSpace(getenv("OPENLIST_PASS")),
		CoverEnabled:  getenvBool("COVER_ENABLED", true),
		StreamEnabled: getenvBool("STREAM_ENABLED", true),
	}

	if ds != nil {
		props := ds.Property(context.Background())
		var err error

		cfg.OpenListBase, err = props.DefaultGet(BaseKey, cfg.OpenListBase)
		if err != nil {
			return err
		}
		cfg.OpenListUser, err = props.DefaultGet(UserKey, cfg.OpenListUser)
		if err != nil {
			return err
		}
		enabled, err := props.DefaultGet(EnabledKey, boolToString(cfg.Enabled))
		if err != nil {
			return err
		}
		cfg.Enabled = parseBool(enabled, cfg.Enabled)

		storedPass, err := props.Get(PassKey)
		if err != nil && !errors.Is(err, model.ErrNotFound) {
			return err
		}
		if err == nil {
			cfg.OpenListPass = strings.TrimSpace(storedPass)
			decryptedPass, decErr := decryptPassword(cfg.OpenListPass)
			if decErr == nil {
				cfg.OpenListPass = decryptedPass
			} else {
				log.Warn("OpenList password is not encrypted, re-encrypting for storage")
				encryptedPass, encErr := encryptPassword(cfg.OpenListPass)
				if encErr != nil {
					log.Warn("Could not encrypt OpenList password", encErr)
				} else if putErr := props.Put(PassKey, encryptedPass); putErr != nil {
					log.Warn("Could not persist encrypted OpenList password", putErr)
				}
			}
		}

		coverEnabled, err := props.DefaultGet(CoverEnabledKey, boolToString(cfg.CoverEnabled))
		if err != nil {
			return err
		}
		streamEnabled, err := props.DefaultGet(StreamEnabledKey, boolToString(cfg.StreamEnabled))
		if err != nil {
			return err
		}
		cfg.CoverEnabled = parseBool(coverEnabled, cfg.CoverEnabled)
		cfg.StreamEnabled = parseBool(streamEnabled, cfg.StreamEnabled)
	}

	cfg.OpenListBase = normalizeBase(cfg.OpenListBase)
	cfg.OpenListUser = strings.TrimSpace(cfg.OpenListUser)
	cfg.OpenListPass = strings.TrimSpace(cfg.OpenListPass)

	stateMu.Lock()
	state = cfg
	tokens.clear()
	stateMu.Unlock()
	return nil
}

func Current() Config {
	stateMu.RLock()
	cfg := state
	stateMu.RUnlock()
	return cfg
}

func Update(ds model.DataStore, input Config) (Config, error) {
	cur := Current()
	next := cur
	next.Enabled = input.Enabled
	next.OpenListBase = normalizeBase(input.OpenListBase)
	next.OpenListUser = strings.TrimSpace(input.OpenListUser)
	if pass := strings.TrimSpace(input.OpenListPass); pass != "" {
		next.OpenListPass = pass
	}
	next.CoverEnabled = input.CoverEnabled
	next.StreamEnabled = input.StreamEnabled

	if ds != nil {
		props := ds.Property(context.Background())
		if err := props.Put(EnabledKey, boolToString(next.Enabled)); err != nil {
			return cur, err
		}
		if err := props.Put(BaseKey, next.OpenListBase); err != nil {
			return cur, err
		}
		if err := props.Put(UserKey, next.OpenListUser); err != nil {
			return cur, err
		}
		encryptedPass, err := encryptPassword(next.OpenListPass)
		if err != nil {
			return cur, err
		}
		if err := props.Put(PassKey, encryptedPass); err != nil {
			return cur, err
		}
		if err := props.Put(CoverEnabledKey, boolToString(next.CoverEnabled)); err != nil {
			return cur, err
		}
		if err := props.Put(StreamEnabledKey, boolToString(next.StreamEnabled)); err != nil {
			return cur, err
		}
	}

	stateMu.Lock()
	state = next
	if credentialsChanged(cur, next) {
		tokens.clear()
	}
	stateMu.Unlock()
	return next, nil
}

func IsConfigured(cfg Config) bool {
	return strings.TrimSpace(cfg.OpenListBase) != "" &&
		strings.TrimSpace(cfg.OpenListUser) != "" &&
		strings.TrimSpace(cfg.OpenListPass) != ""
}

func credentialsChanged(a, b Config) bool {
	return a.OpenListBase != b.OpenListBase ||
		a.OpenListUser != b.OpenListUser ||
		a.OpenListPass != b.OpenListPass
}

func encryptPassword(pass string) (string, error) {
	pass = strings.TrimSpace(pass)
	if pass == "" {
		return "", nil
	}
	return utils.Encrypt(context.Background(), openListEncKey(), pass)
}

func decryptPassword(pass string) (string, error) {
	pass = strings.TrimSpace(pass)
	if pass == "" {
		return "", nil
	}
	return utils.Decrypt(context.Background(), openListEncKey(), pass)
}

func openListEncKey() []byte {
	key := cmp.Or(strings.TrimSpace(conf.Server.PasswordEncryptionKey), consts.DefaultEncryptionKey)
	sum := sha256.Sum256([]byte(key))
	return sum[:]
}
