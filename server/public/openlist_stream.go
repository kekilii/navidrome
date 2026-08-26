package public

import (
	"net/http"

	"github.com/navidrome/navidrome/core/openlist"
	"github.com/navidrome/navidrome/log"
)

func (pub *Router) tryRedirectOpenListSharedStream(w http.ResponseWriter, r *http.Request, songID string) bool {
	if pub == nil || pub.ds == nil {
		return false
	}

	target, err := openlist.ResolveStreamRawURLBySongID(r.Context(), pub.ds, songID)
	if err != nil {
		log.Debug(r.Context(), "OpenList shared stream resolve failed", "id", songID, err)
		return false
	}
	if target == "" {
		return false
	}

	http.Redirect(w, r, target, http.StatusFound)
	return true
}
