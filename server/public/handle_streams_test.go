package public

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/go-chi/jwtauth/v5"
	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/core/openlist"
	"github.com/navidrome/navidrome/core/stream"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("handleStream", func() {
	var ds *tests.MockDataStore
	var mediaRepo *tests.MockMediaFileRepo
	var streamToken string
	var libraryRoot string

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

		libraryRoot = GinkgoT().TempDir()
		trackPath := filepath.Join(libraryRoot, "Artist", "Album", "track.flac")
		Expect(os.MkdirAll(filepath.Dir(trackPath), 0o755)).To(Succeed())
		Expect(os.WriteFile(trackPath, []byte("dummy-audio"), 0o600)).To(Succeed())

		mediaRepo = tests.CreateMockMediaFileRepo()
		mediaRepo.SetData(model.MediaFiles{
			{
				ID:          "song-1",
				Title:       "track",
				Suffix:      "flac",
				BitRate:     320,
				Duration:    180,
				UpdatedAt:   time.Now(),
				Path:        "Artist/Album/track.flac",
				LibraryPath: libraryRoot,
			},
		})
		ds = &tests.MockDataStore{MockedMediaFile: mediaRepo}
		Expect(openlist.Bootstrap(ds)).To(Succeed())

		auth.TokenAuth = jwtauth.New("HS256", []byte("public-secret"), nil)
		var err error
		streamToken, err = auth.CreatePublicToken(auth.Claims{
			ID:     "song-1",
			Format: "raw",
		})
		Expect(err).ToNot(HaveOccurred())
	})

	It("redirects to openlist raw url when openlist resolve succeeds", func() {
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

		streamer := &countingStreamer{err: errors.New("streamer should not be called")}
		router := &Router{ds: ds, streamer: streamer}
		w := httptest.NewRecorder()
		r := newPublicStreamRequest(streamToken)

		router.handleStream(w, r)

		Expect(w.Code).To(Equal(http.StatusFound))
		Expect(w.Header().Get("Location")).To(Equal("http://openlist.local/d/Artist/Album/track.flac"))
		Expect(streamer.called).To(BeFalse())
	})

	It("falls back to streamer when openlist resolve fails", func() {
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
						"code":    500,
						"message": "not found",
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

		streamer := &countingStreamer{delegate: stream.NewMediaStreamer(ds, nil, nil)}
		router := &Router{ds: ds, streamer: streamer}
		w := httptest.NewRecorder()
		r := newPublicStreamRequest(streamToken)

		router.handleStream(w, r)

		Expect(w.Code).To(Equal(http.StatusOK))
		Expect(streamer.called).To(BeTrue())
		Expect(w.Body.String()).To(Equal("dummy-audio"))
	})

	It("returns 500 when fallback streamer fails", func() {
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
						"code":    500,
						"message": "not found",
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

		streamer := &countingStreamer{err: errors.New("fallback failed")}
		router := &Router{ds: ds, streamer: streamer}
		w := httptest.NewRecorder()
		r := newPublicStreamRequest(streamToken)

		Expect(func() {
			router.handleStream(w, r)
		}).ToNot(Panic())
		Expect(w.Code).To(Equal(http.StatusInternalServerError))
		Expect(streamer.called).To(BeTrue())
	})
})

func newPublicStreamRequest(token string) *http.Request {
	params := url.Values{}
	params.Set(":id", token)
	return httptest.NewRequest(http.MethodGet, "/s?"+params.Encode(), nil)
}

type countingStreamer struct {
	delegate stream.MediaStreamer
	called   bool
	err      error
}

func (s *countingStreamer) NewStream(ctx context.Context, mf *model.MediaFile, req stream.Request) (*stream.Stream, error) {
	s.called = true
	if s.err != nil {
		return nil, s.err
	}
	if s.delegate == nil {
		return nil, errors.New("missing streamer delegate")
	}
	return s.delegate.NewStream(ctx, mf, req)
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func jsonResponse(payload any) *http.Response {
	body, _ := json.Marshal(payload)
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(bytes.NewReader(body)),
	}
}
