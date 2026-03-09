package openlist

import (
	"context"
	"strings"

	"github.com/navidrome/navidrome/model"
)

// ResolveStreamRawURLBySongID resolves an OpenList raw URL for the given song ID.
// It returns an empty URL with nil error when OpenList streaming is not applicable.
func ResolveStreamRawURLBySongID(ctx context.Context, ds model.DataStore, songID string) (string, error) {
	cfg := Current()
	if !cfg.Enabled || !cfg.StreamEnabled || !IsConfigured(cfg) {
		return "", nil
	}

	songID = strings.TrimSpace(songID)
	if songID == "" || ds == nil {
		return "", nil
	}

	song, err := ds.MediaFile(ctx).Get(songID)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(song.LibraryPath) == "" {
		return "", nil
	}

	openListPath := BuildOpenListPath(song.Path, song.LibraryPath)
	if openListPath == "" {
		return "", nil
	}
	return ResolveRawURL(ctx, openListPath)
}
