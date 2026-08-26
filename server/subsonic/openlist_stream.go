package subsonic

import (
	"net/http"

	"github.com/navidrome/navidrome/core/openlist"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/utils/req"
)

func (api *Router) tryRedirectOpenListStream(w http.ResponseWriter, r *http.Request, id string) bool {
	if api == nil || api.ds == nil {
		return false
	}
	target, err := openlist.ResolveStreamRawURLBySongID(r.Context(), api.ds, id)
	if err != nil || target == "" {
		return false
	}
	if mf, err := api.ds.MediaFile(r.Context()).Get(id); err != nil {
		log.Error(r.Context(), "Error retrieving OpenList media file for playback log", "id", id, err)
	} else if mf != nil {
		logNowPlaying(r.Context(), mf, req.Params(r).IntOr("timeOffset", 0))
	}
	http.Redirect(w, r, target, http.StatusFound) //nolint:gosec // OpenList target is resolved from configured server paths
	return true
}
