package nativeapi

import (
	"bytes"
	"cmp"
	"context"
	"crypto/sha256"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/auth"
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
		ds = &tests.MockDataStore{
			MockedUser:     userRepo,
			MockedProperty: props,
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
})

func openListEncKey() []byte {
	key := cmp.Or(conf.Server.PasswordEncryptionKey, consts.DefaultEncryptionKey)
	sum := sha256.Sum256([]byte(key))
	return sum[:]
}
