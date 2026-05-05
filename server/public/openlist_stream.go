package public

import (
	"context"

	"github.com/navidrome/navidrome/core/openlist"
)

func (pub *Router) resolveOpenListSharedStreamTarget(ctx context.Context, songID string) (string, error) {
	if pub == nil || pub.ds == nil {
		return "", nil
	}
	return openlist.ResolveStreamRawURLBySongID(ctx, pub.ds, songID)
}
