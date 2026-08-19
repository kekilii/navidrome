# OpenList Merge Isolation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove avoidable OpenList/upstream merge overlap while retaining the established routing, authorization, deadline, and fallback behavior.

**Architecture:** OpenList owns its persisted-property keys in `core/openlist`; their string values do not change. OpenList-specific tests live beside their dedicated redirect helpers, leaving upstream handler tests focused on upstream behavior. The existing one-line production seams stay in shared handlers because they enforce the required ordering.

**Tech Stack:** Go, Ginkgo/Gomega, Go modules.

---

## File structure

- `consts/consts.go` — upstream-owned common constants only; remove all six OpenList keys.
- `core/openlist/openlist.go` — declares and uses OpenList persisted-property keys.
- `core/openlist/openlist_test.go` — verifies exact stable key values and Bootstrap/Update persistence.
- `server/nativeapi/openlist_test.go` — refers to OpenList-owned keys, not `consts`.
- `server/public/handle_streams_test.go` — keeps only generic public-stream tests.
- `server/public/openlist_stream_test.go` — owns the public-share OpenList redirect suite and local test doubles.
- `server/subsonic/media_retrieval_test.go` — keeps generic cover/lyrics behavior and its generic `fakeArtwork`.
- `server/subsonic/openlist_cover_test.go` — owns OpenList cover redirect, fallback, deadline, and resolver-seam tests with an independent artwork fake.

### Task 1: Move persisted OpenList property keys into the owning package

**Files:**
- Modify: `consts/consts.go:23-28`
- Modify: `core/openlist/openlist.go:18-151`
- Create: `core/openlist/openlist_test.go`
- Modify: `server/nativeapi/openlist_test.go:13-173`

- [ ] **Step 1: Write a failing OpenList key compatibility test**

Create `core/openlist/openlist_test.go` with a standard-library test for the persisted strings:

```go
package openlist

import "testing"

func TestPersistedPropertyKeys(t *testing.T) {
    keys := map[string]string{
        "EnabledKey":       EnabledKey,
        "BaseKey":          BaseKey,
        "UserKey":          UserKey,
        "PassKey":          PassKey,
        "CoverEnabledKey":  CoverEnabledKey,
        "StreamEnabledKey": StreamEnabledKey,
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
    }
}
```

- [ ] **Step 2: Verify the test fails before declaring the keys**

Run: `go test ./core/openlist`

Expected: compile failure because the six `openlist.*Key` identifiers do not yet exist.

- [ ] **Step 3: Declare the six keys in `core/openlist` and replace every reference**

Add this block immediately below `RecordID` in `core/openlist/openlist.go`:

```go
const (
    RecordID          = "openlist"
    EnabledKey        = "OpenListEnabled"
    BaseKey           = "OpenListBase"
    UserKey           = "OpenListUser"
    PassKey           = "OpenListPass"
    CoverEnabledKey   = "OpenListCoverEnabled"
    StreamEnabledKey  = "OpenListStreamEnabled"
)
```

Replace every `consts.OpenList…Key` in `Bootstrap` and `Update` with the matching package identifier. Retain `consts` only for `DefaultEncryptionKey`. Remove the six OpenList declarations from `consts/consts.go` without changing any other constants.

In `server/nativeapi/openlist_test.go`, keep its existing `core/openlist` import and replace all `consts.OpenList…Key` map/index uses with the corresponding `openlist.*Key`; retain `consts.UIAuthorizationHeader` unchanged.

- [ ] **Step 4: Format and verify persisted configuration compatibility**

Run: `gofmt -w consts/consts.go core/openlist/openlist.go core/openlist/openlist_test.go server/nativeapi/openlist_test.go && go test ./core/openlist ./server/nativeapi`

Expected: PASS. `rg -n 'OpenList(?:Enabled|Base|User|Pass|CoverEnabled|StreamEnabled)Key' consts` produces no output.

- [ ] **Step 5: Commit the isolated property-key ownership**

```bash
git add consts/consts.go core/openlist/openlist.go core/openlist/openlist_test.go server/nativeapi/openlist_test.go
git commit -m "refactor(openlist): own persisted property keys"
```

### Task 2: Move public-share OpenList tests beside their helper

**Files:**
- Modify: `server/public/handle_streams_test.go:1-683`
- Create: `server/public/openlist_stream_test.go`

- [ ] **Step 1: Copy the OpenList suite into the dedicated test file without changing assertions**

Create `server/public/openlist_stream_test.go` in package `public`. Move verbatim the complete `Describe("handleStream OpenList", …)` block currently at lines 257-645 and its five dedicated support declarations at lines 647-683. Do not change test names, setup values, request paths, expectations, or helper bodies while relocating them.

Carry only the imports that suite needs: `bytes`, `context`, `encoding/json`, `errors`, `io`, `net/http`, `net/http/httptest`, `net/url`, `os`, `path/filepath`, `time`, `jwtauth`, `core/auth`, `core/openlist`, `core/stream`, `model`, `tests`, and Ginkgo/Gomega. Preserve its cleanup that restores all six environment variables and calls `openlist.Bootstrap(nil)`, and each `DeferCleanup(restoreClient)`.

- [ ] **Step 2: Remove the copied suite and prune imports from the upstream-owned test**

Delete the exact `Describe("handleStream OpenList", …)` block and the five support declarations from `handle_streams_test.go`. Remove only imports now made unused by that deletion; do not alter the generic `Describe("handleStream", …)` suite or its invalid share `400` assertion.

- [ ] **Step 3: Run public tests to verify the relocated suite still exercises the handler**

Run: `gofmt -w server/public/handle_streams_test.go server/public/openlist_stream_test.go && go test ./server/public`

Expected: PASS. The relocated suite still proves redirect success, streamer fallback, fallback failure, no OpenList request for invalid/deleted/expired/missing-media requests, missing library access, and a share that excludes the requested song.

- [ ] **Step 4: Confirm test ownership is separated**

Run: `rg -n 'handleStream OpenList|OpenList shared stream' server/public/handle_streams_test.go server/public/openlist_stream_test.go`

Expected: the OpenList suite is reported only in `server/public/openlist_stream_test.go`.

- [ ] **Step 5: Commit the public test relocation**

```bash
git add server/public/handle_streams_test.go server/public/openlist_stream_test.go
git commit -m "test(openlist): isolate public stream coverage"
```

### Task 3: Move Subsonic cover OpenList tests beside their helper

**Files:**
- Modify: `server/subsonic/media_retrieval_test.go:1-533`
- Create: `server/subsonic/openlist_cover_test.go`

- [ ] **Step 1: Create a self-contained OpenList-cover suite**

Create `server/subsonic/openlist_cover_test.go` in package `subsonic`. It must contain a `Describe("OpenList cover redirects", …)` suite with independent datastore, album repository, `coverTestArtwork`, and recorder setup. Its `BeforeEach` must create:

```go
ds = &tests.MockDataStore{
    MockedMediaFile: mediaRepo,
    MockedAlbum:     albumRepo,
    MockedRadio:     radioRepo,
}
router = New(ds, artwork, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, lyrics.NewLyrics(ds, nil), nil, nil)
```

Copy the existing environment snapshot/restore protocol: record each of `OPENLIST_BASE`, `OPENLIST_USER`, `OPENLIST_PASS`, `OPENLIST_ENABLED`, `COVER_ENABLED`, and `STREAM_ENABLED`; unset them for the test; restore exact presence/value in `DeferCleanup`; then call `openlist.Bootstrap(nil)`. The suite also restores every test HTTP client with `DeferCleanup`.

Move these exact behavioral tests from the generic cover suite, keeping the current request paths and expectations:

- `redirects to openlist when mf cover is available`;
- `redirects to openlist when al cover is available`;
- `falls back to existing artwork when openlist cover lookup fails`;
- `uses the artwork lookup deadline for the OpenList cover lookup`;
- `OpenList cover resolver seam` returns empty target without datastore/configuration.

`coverTestArtwork.GetOrPlaceholder` returns an `*artwork.Image` made from `io.NopCloser(bytes.NewReader([]byte(c.data)))`, records `recvID`, and never shares the generic `fakeArtwork` type.

- [ ] **Step 2: Prove the copied dedicated suite runs before removal**

Run: `go test ./server/subsonic -ginkgo.focus='OpenList cover' -count=1`

Expected: PASS with the copied tests, demonstrating the dedicated fixture has no dependency on `media_retrieval_test.go` globals.

- [ ] **Step 3: Remove OpenList setup and tests from the generic media retrieval test**

Delete the six-variable environment cleanup and `openlist.Bootstrap(ds)` from `MediaRetrievalController` setup. Remove its `core/openlist` import and OpenList-only HTTP/OS imports if unused afterwards. Delete only the four OpenList cover tests listed above, `coverRoundTripper`, and the trailing resolver-seam `Describe`; retain all generic cover, cancellation, cache-header, lyrics, `fakeArtwork`, and `mockedMediaFile` tests unchanged.

- [ ] **Step 4: Format and run all Subsonic coverage**

Run: `gofmt -w server/subsonic/media_retrieval_test.go server/subsonic/openlist_cover_test.go && go test ./server/subsonic`

Expected: PASS. Verify `rg -n 'OpenList cover|openlist\.' server/subsonic/media_retrieval_test.go` produces no output.

- [ ] **Step 5: Commit the cover test relocation**

```bash
git add server/subsonic/media_retrieval_test.go server/subsonic/openlist_cover_test.go
git commit -m "test(openlist): isolate cover redirect coverage"
```

### Task 4: Verify future-merge boundaries

**Files:**
- Verify: `consts/consts.go`
- Verify: `server/nativeapi/native_api.go`
- Verify: `server/public/handle_streams.go`
- Verify: `server/subsonic/media_retrieval.go`
- Verify: `server/subsonic/stream.go`
- Verify: `docs/superpowers/specs/2026-08-19-openlist-merge-isolation-design.md`

- [ ] **Step 1: Confirm only the intentional production seams remain**

Run:

```bash
rg -n 'BootstrapWithLog|tryRedirectOpenListSharedStream|tryRedirectOpenListCover|tryRedirectOpenListStream' \
  server/nativeapi/native_api.go server/public/handle_streams.go \
  server/subsonic/media_retrieval.go server/subsonic/stream.go
```

Expected: one `BootstrapWithLog` call, one public-stream delegation, one cover delegation, and one Subsonic-stream delegation. Do not introduce a plugin framework or middleware abstraction.

- [ ] **Step 2: Run the complete targeted regression suite**

Run: `go test ./core/openlist ./server/nativeapi ./server/public ./server/subsonic && git diff --check`

Expected: all packages PASS and no whitespace errors.

- [ ] **Step 3: Review the merge-conflict surface and commit verification notes only if changed**

Run: `git diff --stat HEAD~3..HEAD -- consts core/openlist server/nativeapi server/public server/subsonic && git status --short`

Expected: `consts/consts.go` has no OpenList-specific changes; OpenList-specific public and cover tests live in dedicated files; working tree is clean. Do not add a documentation-only commit if no documentation changed.
