package openlist_test

import (
	"testing"

	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/openlist"
)

func TestPersistedPropertyKeys(t *testing.T) {
	keys := map[string]string{
		"EnabledKey":       openlist.EnabledKey,
		"BaseKey":          openlist.BaseKey,
		"UserKey":          openlist.UserKey,
		"PassKey":          openlist.PassKey,
		"CoverEnabledKey":  openlist.CoverEnabledKey,
		"StreamEnabledKey": openlist.StreamEnabledKey,
	}
	deprecatedKeys := map[string]string{
		"EnabledKey":       consts.OpenListEnabledKey,
		"BaseKey":          consts.OpenListBaseKey,
		"UserKey":          consts.OpenListUserKey,
		"PassKey":          consts.OpenListPassKey,
		"CoverEnabledKey":  consts.OpenListCoverEnabledKey,
		"StreamEnabledKey": consts.OpenListStreamEnabledKey,
	}
	want := map[string]string{
		"EnabledKey":       "OpenListEnabled",
		"BaseKey":          "OpenListBase",
		"UserKey":          "OpenListUser",
		"PassKey":          "OpenListPass",
		"CoverEnabledKey":  "OpenListCoverEnabled",
		"StreamEnabledKey": "OpenListStreamEnabled",
	}
	for name, got := range keys {
		if got != want[name] {
			t.Errorf("%s = %q, want %q", name, got, want[name])
		}
		if deprecated := deprecatedKeys[name]; deprecated != got {
			t.Errorf("deprecated consts.%s = %q, want canonical value %q", name, deprecated, got)
		} else if deprecated != want[name] {
			t.Errorf("deprecated consts.%s = %q, want %q", name, deprecated, want[name])
		}
	}
}
