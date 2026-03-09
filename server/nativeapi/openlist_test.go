package nativeapi

import (
	"bytes"
	"cmp"
	"context"
	"crypto/sha256"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/core/openlist"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server"
	"github.com/navidrome/navidrome/tests"
	"github.com/navidrome/navidrome/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("OpenList API", func() {
	var ds *tests.MockDataStore
	var router http.Handler
	var adminUser, regularUser model.User
	var props *tests.MockedPropertyRepo
	var mediaRepo *tests.MockMediaFileRepo

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		conf.Server.DevUIShowConfig = true
		for _, key := range []string{
			"OPENLIST_BASE",
			"OPENLIST_USER",
			"OPENLIST_PASS",
			"OPENLIST_ENABLED",
			"COVER_ENABLED",
			"STREAM_ENABLED",
		} {
			Expect(os.Unsetenv(key)).To(Succeed())
		}

		props = &tests.MockedPropertyRepo{}
		userRepo := tests.CreateMockUserRepo()
		mediaRepo = tests.CreateMockMediaFileRepo()
		mediaRepo.SetData(model.MediaFiles{
			{
				ID:          "song-1",
				Path:        "Artist/Album/track.flac",
				LibraryPath: "/music",
			},
		})
		ds = &tests.MockDataStore{
			MockedMediaFile: mediaRepo,
			MockedUser:      userRepo,
			MockedProperty:  props,
		}
		auth.Init(ds)
		nativeRouter := New(ds, nil, nil, nil, tests.NewMockLibraryService(), tests.NewMockUserService(), nil, nil)
		router = server.JWTVerifier(nativeRouter)

		adminUser = model.User{
			ID:          "admin-ol",
			UserName:    "admin-openlist",
			Name:        "Admin OpenList",
			IsAdmin:     true,
			NewPassword: "adminpass",
		}
		regularUser = model.User{
			ID:          "user-ol",
			UserName:    "user-openlist",
			Name:        "User OpenList",
			IsAdmin:     false,
			NewPassword: "userpass",
		}
		Expect(ds.User(context.TODO()).Put(&adminUser)).To(Succeed())
		Expect(ds.User(context.TODO()).Put(&regularUser)).To(Succeed())
	})

	It("allows admin to get openlist config", func() {
		props.Data = map[string]string{
			consts.OpenListEnabledKey:       "true",
			consts.OpenListBaseKey:          "http://127.0.0.1:5244",
			consts.OpenListUserKey:          "admin",
			consts.OpenListPassKey:          "secret-pass",
			consts.OpenListCoverEnabledKey:  "true",
			consts.OpenListStreamEnabledKey: "false",
		}
		nativeRouter := New(ds, nil, nil, nil, tests.NewMockLibraryService(), tests.NewMockUserService(), nil, nil)
		router = server.JWTVerifier(nativeRouter)

		token, err := auth.CreateToken(&adminUser)
		Expect(err).ToNot(HaveOccurred())

		req := httptest.NewRequest(http.MethodGet, "/openlist/openlist", nil)
		req.Header.Set(consts.UIAuthorizationHeader, "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		Expect(w.Code).To(Equal(http.StatusOK))
		var payload openListConfigPayload
		Expect(json.Unmarshal(w.Body.Bytes(), &payload)).To(Succeed())
		Expect(payload.ID).To(Equal("openlist"))
		Expect(payload.Enabled).To(BeTrue())
		Expect(payload.OpenListBase).To(Equal("http://127.0.0.1:5244"))
		Expect(payload.OpenListUser).To(Equal("admin"))
		Expect(payload.OpenListPass).To(Equal(""))
		Expect(payload.CoverEnabled).To(BeTrue())
		Expect(payload.StreamEnabled).To(BeFalse())
	})

	It("denies non-admin access", func() {
		token, err := auth.CreateToken(&regularUser)
		Expect(err).ToNot(HaveOccurred())

		req := httptest.NewRequest(http.MethodGet, "/openlist/openlist", nil)
		req.Header.Set(consts.UIAuthorizationHeader, "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		Expect(w.Code).To(Equal(http.StatusForbidden))
	})

	It("keeps existing password when put payload has empty password", func() {
		props.Data = map[string]string{
			consts.OpenListEnabledKey:       "true",
			consts.OpenListBaseKey:          "http://openlist.local:5244",
			consts.OpenListUserKey:          "existing-user",
			consts.OpenListPassKey:          "existing-pass",
			consts.OpenListCoverEnabledKey:  "true",
			consts.OpenListStreamEnabledKey: "true",
		}
		nativeRouter := New(ds, nil, nil, nil, tests.NewMockLibraryService(), tests.NewMockUserService(), nil, nil)
		router = server.JWTVerifier(nativeRouter)

		token, err := auth.CreateToken(&adminUser)
		Expect(err).ToNot(HaveOccurred())

		body, _ := json.Marshal(openListConfigPayload{
			ID:            "openlist",
			Enabled:       true,
			OpenListBase:  "http://openlist.updated:5244",
			OpenListUser:  "updated-user",
			OpenListPass:  "",
			CoverEnabled:  false,
			StreamEnabled: true,
		})
		req := httptest.NewRequest(http.MethodPut, "/openlist/openlist", bytes.NewReader(body))
		req.Header.Set(consts.UIAuthorizationHeader, "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		Expect(w.Code).To(Equal(http.StatusOK))
		var payload openListConfigPayload
		Expect(json.Unmarshal(w.Body.Bytes(), &payload)).To(Succeed())
		Expect(payload.OpenListPass).To(Equal(""))

		Expect(props.Data[consts.OpenListBaseKey]).To(Equal("http://openlist.updated:5244"))
		Expect(props.Data[consts.OpenListUserKey]).To(Equal("updated-user"))
		Expect(props.Data[consts.OpenListEnabledKey]).To(Equal("true"))
		Expect(props.Data[consts.OpenListPassKey]).ToNot(Equal("existing-pass"))
		decryptedPass, err := utils.Decrypt(context.Background(), openListEncKey(), props.Data[consts.OpenListPassKey])
		Expect(err).ToNot(HaveOccurred())
		Expect(decryptedPass).To(Equal("existing-pass"))
		Expect(props.Data[consts.OpenListCoverEnabledKey]).To(Equal("false"))
		Expect(props.Data[consts.OpenListStreamEnabledKey]).To(Equal("true"))
	})

	It("allows non-admin user to resolve stream raw url", func() {
		restoreClient := openlist.SetHTTPClientForTests(&http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				switch req.URL.Path {
				case "/api/auth/login":
					return jsonResponse(map[string]any{
						"code": 200,
						"data": map[string]any{"token": "openlist-token"},
					}), nil
				case "/api/fs/get":
					return jsonResponse(map[string]any{
						"code": 200,
						"data": map[string]any{
							"raw_url": "/d/Artist/Album/track.flac",
							"is_dir":  false,
						},
					}), nil
				default:
					return jsonResponse(map[string]any{
						"code":    404,
						"message": "not found",
					}), nil
				}
			}),
		})
		DeferCleanup(restoreClient)

		_, err := openlist.Update(ds, openlist.Config{
			Enabled:       true,
			OpenListBase:  "http://openlist.local",
			OpenListUser:  "admin",
			OpenListPass:  "secret",
			CoverEnabled:  true,
			StreamEnabled: true,
		})
		Expect(err).ToNot(HaveOccurred())

		token, err := auth.CreateToken(&regularUser)
		Expect(err).ToNot(HaveOccurred())

		req := httptest.NewRequest(http.MethodGet, "/openlist/stream/song-1", nil)
		req.Header.Set(consts.UIAuthorizationHeader, "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		Expect(w.Code).To(Equal(http.StatusOK))
		var payload openListStreamPayload
		Expect(json.Unmarshal(w.Body.Bytes(), &payload)).To(Succeed())
		Expect(payload.RawURL).To(Equal("http://openlist.local/d/Artist/Album/track.flac"))
	})

	It("returns empty raw url when resolve fails", func() {
		_, err := openlist.Update(ds, openlist.Config{
			Enabled:       true,
			OpenListBase:  "http://127.0.0.1:1",
			OpenListUser:  "admin",
			OpenListPass:  "secret",
			CoverEnabled:  true,
			StreamEnabled: true,
		})
		Expect(err).ToNot(HaveOccurred())

		token, err := auth.CreateToken(&regularUser)
		Expect(err).ToNot(HaveOccurred())

		req := httptest.NewRequest(http.MethodGet, "/openlist/stream/song-1", nil)
		req.Header.Set(consts.UIAuthorizationHeader, "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		Expect(w.Code).To(Equal(http.StatusOK))
		var payload openListStreamPayload
		Expect(json.Unmarshal(w.Body.Bytes(), &payload)).To(Succeed())
		Expect(payload.RawURL).To(BeEmpty())
	})
})

func openListEncKey() []byte {
	key := cmp.Or(conf.Server.PasswordEncryptionKey, consts.DefaultEncryptionKey)
	sum := sha256.Sum256([]byte(key))
	return sum[:]
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func jsonResponse(payload any) *http.Response {
	data, _ := json.Marshal(payload)
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewReader(data)),
		Header:     make(http.Header),
	}
}
