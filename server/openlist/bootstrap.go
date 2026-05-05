package openlist

import (
	coreopenlist "github.com/navidrome/navidrome/core/openlist"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

// BootstrapWithLog initializes OpenList state from the datastore and emits a
// consistent warning message if bootstrap fails.
func BootstrapWithLog(ds model.DataStore) {
	if err := coreopenlist.Bootstrap(ds); err != nil {
		log.Warn("Could not bootstrap OpenList settings", err)
	}
}
