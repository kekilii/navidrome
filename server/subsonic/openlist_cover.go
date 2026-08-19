package subsonic

import (
	"context"
	"net/http"
	"strings"

	"github.com/navidrome/navidrome/core/openlist"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server/filter"
)

func (api *Router) tryRedirectOpenListCover(w http.ResponseWriter, r *http.Request, id string) bool {
	target, err := api.resolveOpenListCoverTarget(r.Context(), id)
	if err != nil || target == "" {
		return false
	}
	http.Redirect(w, r, target, http.StatusFound) //nolint:gosec // OpenList target is resolved from configured server paths
	return true
}

func (api *Router) resolveOpenListCoverTarget(ctx context.Context, id string) (string, error) {
	if api == nil || api.ds == nil {
		return "", nil
	}

	cfg := openlist.Current()
	if !cfg.Enabled || !cfg.CoverEnabled || !openlist.IsConfigured(cfg) {
		return "", nil
	}

	artworkID, parseErr := model.ParseArtworkID(strings.TrimSpace(id))
	if parseErr != nil {
		return "", parseErr
	}

	var song *model.MediaFile
	var err error
	switch artworkID.Kind {
	case model.KindMediaFileArtwork:
		song, err = api.ds.MediaFile(ctx).Get(artworkID.ID)
	case model.KindAlbumArtwork:
		songs, getErr := api.ds.MediaFile(ctx).GetAll(filter.SongsByAlbum(artworkID.ID))
		if getErr != nil {
			return "", getErr
		}
		if len(songs) == 0 {
			return "", model.ErrNotFound
		}
		song = &songs[0]
	default:
		return "", nil
	}
	if err != nil {
		return "", err
	}

	if strings.TrimSpace(song.LibraryPath) == "" {
		return "", nil
	}

	coverPath := openlist.BuildCoverPath(song.Path)
	if coverPath == "" {
		return "", nil
	}
	openListPath := openlist.BuildOpenListPath(coverPath, song.LibraryPath)
	if openListPath == "" {
		return "", nil
	}
	return openlist.ResolveRawURL(ctx, openListPath)
}
