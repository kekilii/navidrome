package subsonic

import (
	"context"

	"github.com/navidrome/navidrome/core/openlist"
)

func (api *Router) resolveOpenListStreamTarget(ctx context.Context, id string) (string, error) {
	if api == nil || api.ds == nil {
		return "", nil
	}
	return openlist.ResolveStreamRawURLBySongID(ctx, api.ds, id)
}
