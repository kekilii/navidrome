package subsonic

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/core/openlist"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Stream OpenList", func() {
	var ds *tests.MockDataStore
	var mediaRepo *tests.MockMediaFileRepo

	BeforeEach(func() {
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

		mediaRepo = tests.CreateMockMediaFileRepo()
		mediaRepo.SetData(model.MediaFiles{
			{
				ID:          "song-1",
				Path:        "Artist/Album/track.flac",
				LibraryPath: "/music",
			},
		})
		ds = &tests.MockDataStore{MockedMediaFile: mediaRepo}
		Expect(openlist.Bootstrap(ds)).To(Succeed())
	})

	It("redirects to openlist raw url when stream proxy succeeds", func() {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/api/auth/login":
				_ = json.NewEncoder(w).Encode(map[string]any{
					"code": 200,
					"data": map[string]any{"token": "openlist-token"},
				})
			case "/api/fs/get":
				_ = json.NewEncoder(w).Encode(map[string]any{
					"code": 200,
					"data": map[string]any{
						"raw_url": "/d/Artist/Album/track.flac",
						"is_dir":  false,
					},
				})
			default:
				http.NotFound(w, r)
			}
		}))
		DeferCleanup(srv.Close)

		_, err := openlist.Update(ds, openlist.Config{
			Enabled:       true,
			OpenListBase:  srv.URL,
			OpenListUser:  "admin",
			OpenListPass:  "secret",
			CoverEnabled:  true,
			StreamEnabled: true,
		})
		Expect(err).ToNot(HaveOccurred())

		streamer := &fakeStreamer{err: errors.New("streamer should not be called")}
		router := &Router{ds: ds, streamer: streamer}
		w := httptest.NewRecorder()
		r := newGetRequest("id=song-1")

		resp, err := router.Stream(w, r)
		Expect(err).ToNot(HaveOccurred())
		Expect(resp).To(BeNil())
		Expect(streamer.called).To(BeFalse())
		Expect(w.Code).To(Equal(http.StatusFound))
		Expect(w.Header().Get("Location")).To(Equal(srv.URL + "/d/Artist/Album/track.flac"))
	})

	It("falls back to streamer when openlist proxy fails", func() {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/api/auth/login":
				_ = json.NewEncoder(w).Encode(map[string]any{
					"code": 200,
					"data": map[string]any{"token": "openlist-token"},
				})
			case "/api/fs/get":
				_ = json.NewEncoder(w).Encode(map[string]any{
					"code":    500,
					"message": "not found",
				})
			default:
				http.NotFound(w, r)
			}
		}))
		DeferCleanup(srv.Close)

		_, err := openlist.Update(ds, openlist.Config{
			Enabled:       true,
			OpenListBase:  srv.URL,
			OpenListUser:  "admin",
			OpenListPass:  "secret",
			CoverEnabled:  true,
			StreamEnabled: true,
		})
		Expect(err).ToNot(HaveOccurred())

		expectedErr := errors.New("fallback streamer called")
		streamer := &fakeStreamer{err: expectedErr}
		router := &Router{ds: ds, streamer: streamer}
		w := httptest.NewRecorder()
		r := newGetRequest("id=song-1")

		resp, err := router.Stream(w, r)
		Expect(resp).To(BeNil())
		Expect(err).To(MatchError(expectedErr))
		Expect(streamer.called).To(BeTrue())
	})
})

type fakeStreamer struct {
	called bool
	err    error
}

func (f *fakeStreamer) NewStream(_ context.Context, _ string, _ string, _ int, _ int) (*core.Stream, error) {
	f.called = true
	return nil, f.err
}

func (f *fakeStreamer) DoStream(_ context.Context, _ *model.MediaFile, _ string, _ int, _ int) (*core.Stream, error) {
	f.called = true
	return nil, f.err
}
