package subsonic

import (
	"net/http"

	"github.com/navidrome/navidrome/core/openlist"
)

func (api *Router) tryRedirectOpenListStream(w http.ResponseWriter, r *http.Request, id string) bool {
	if api == nil || api.ds == nil {
		return false
	}
	target, err := openlist.ResolveStreamRawURLBySongID(r.Context(), api.ds, id)
	if err != nil || target == "" {
		return false
	}
	http.Redirect(w, r, target, http.StatusFound) //nolint:gosec // OpenList target is resolved from configured server paths
	return true
}
