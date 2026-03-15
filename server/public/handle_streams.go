package public

import (
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/core/openlist"
	"github.com/navidrome/navidrome/core/stream"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/req"
)

func (pub *Router) handleStream(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	p := req.Params(r)
	tokenId, _ := p.String(":id")
	info, err := decodeStreamInfo(tokenId)
	if err != nil {
		log.Error(ctx, "Error parsing shared stream info", err)
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	target, err := openlist.ResolveStreamRawURLBySongID(ctx, pub.ds, info.id)
	if err != nil {
		log.Debug(ctx, "OpenList shared stream resolve failed", "id", info.id, err)
	} else if target != "" {
		http.Redirect(w, r, target, http.StatusFound)
		return
	}

	mf, err := pub.ds.MediaFile(ctx).Get(info.id)
	switch {
	case errors.Is(err, model.ErrNotFound):
		log.Warn(ctx, "Shared stream media file not found", "id", info.id, err)
		http.Error(w, "file not found", http.StatusNotFound)
		return
	case err != nil:
		log.Error(ctx, "Error loading shared stream media file", "id", info.id, err)
		http.Error(w, "invalid request", http.StatusInternalServerError)
		return
	}

	mediaStream, err := pub.streamer.NewStream(ctx, mf, stream.Request{Format: info.format, BitRate: info.bitrate})
	if err != nil {
		log.Error(ctx, "Error starting shared stream", err)
		http.Error(w, "invalid request", http.StatusInternalServerError)
		return
	}

	// Make sure the stream will be closed at the end, to avoid leakage
	defer func() {
		if err := mediaStream.Close(); err != nil && log.IsGreaterOrEqualTo(log.LevelDebug) {
			log.Error("Error closing shared stream", "id", info.id, "file", mediaStream.Name(), err)
		}
	}()

	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Content-Duration", strconv.FormatFloat(float64(mediaStream.Duration()), 'G', -1, 32))

	if mediaStream.Seekable() {
		http.ServeContent(w, r, mediaStream.Name(), mediaStream.ModTime(), mediaStream)
	} else {
		// If the stream doesn't provide a size (i.e. is not seekable), we can't support ranges/content-length
		w.Header().Set("Accept-Ranges", "none")
		w.Header().Set("Content-Type", mediaStream.ContentType())

		estimateContentLength := p.BoolOr("estimateContentLength", false)

		// if Client requests the estimated content-length, send it
		if estimateContentLength {
			length := strconv.Itoa(mediaStream.EstimatedContentLength())
			log.Trace(ctx, "Estimated content-length", "contentLength", length)
			w.Header().Set("Content-Length", length)
		}

		if r.Method == http.MethodHead {
			go func() { _, _ = io.Copy(io.Discard, mediaStream) }()
		} else {
			c, err := io.Copy(w, mediaStream)
			if log.IsGreaterOrEqualTo(log.LevelDebug) {
				if err != nil {
					log.Error(ctx, "Error sending shared transcoded file", "id", info.id, err)
				} else {
					log.Trace(ctx, "Success sending shared transcode file", "id", info.id, "size", c)
				}
			}
		}
	}
}

type shareTrackInfo struct {
	id      string
	format  string
	bitrate int
}

func decodeStreamInfo(tokenString string) (shareTrackInfo, error) {
	token, err := auth.TokenAuth.Decode(tokenString)
	if err != nil {
		return shareTrackInfo{}, err
	}
	if token == nil {
		return shareTrackInfo{}, errors.New("unauthorized")
	}
	c := auth.ClaimsFromToken(token)
	if c.ID == "" {
		return shareTrackInfo{}, errors.New("required claim \"id\" not found")
	}
	return shareTrackInfo{
		id:      c.ID,
		format:  c.Format,
		bitrate: c.BitRate,
	}, nil
}
