# OpenList Upstream Merge Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Finish the upstream merge with OpenList configuration, cover art, and stream redirects working, while keeping custom behavior outside upstream-owned handlers wherever possible.

**Architecture:** Retain upstream implementations in shared Navidrome files. Route OpenList redirects through dedicated `openlist_*.go` helpers that return whether they handled the response; shared stream and cover handlers make one delegation call after their normal authorization/validation boundary. Native API bootstrap remains a narrow registration seam, and property constants retain both branches' declarations.

**Tech Stack:** Go, chi HTTP routing, Ginkgo/Gomega, Go modules, React/Vitest.

---

## File structure

- `consts/consts.go` — shared persisted-property names; combine upstream maintenance/artwork keys with OpenList keys.
- `server/nativeapi/native_api.go` — construct the upstream artwork uploader router and invoke only the OpenList bootstrap seam.
- `server/public/handle_streams.go` — preserve upstream token/share authorization and invoke a single dedicated OpenList redirect helper after authorization.
- `server/public/openlist_stream.go` — own shared-link OpenList resolution, logging, and redirect response.
- `server/subsonic/media_retrieval.go` — preserve the upstream `artwork.Image` API and invoke one cover redirect helper before normal artwork retrieval.
- `server/subsonic/openlist_cover.go` — own OpenList cover lookup plus redirect behavior.
- `server/subsonic/stream.go` — invoke one OpenList stream redirect helper before normal streamer construction.
- `server/subsonic/openlist_stream.go` — own OpenList stream lookup plus redirect behavior.
- `server/{public,subsonic,nativeapi}/*_test.go` — retain upstream expectations and cover OpenList success/fallback/authorization behavior.

### Task 1: Resolve constants and native API bootstrap conflict

**Files:**
- Modify: `consts/consts.go:20-36`
- Modify: `server/nativeapi/native_api.go:18-56`
- Test: `server/nativeapi/openlist_test.go:264-290`

- [ ] **Step 1: Preserve the bootstrap-route contract in a focused test**

Keep `OpenList bootstrap seam` in `server/nativeapi/openlist_test.go`. Its existing request must receive `200 OK` for authenticated administrators:

```go
router := server.JWTVerifier(New(ds, nil, nil, nil, tests.NewMockLibraryService(), tests.NewMockUserService(), nil, nil, nil))
router.ServeHTTP(w, req)
Expect(w.Code).To(Equal(http.StatusOK))
```

- [ ] **Step 2: Run the focused test before resolving the conflict**

Run: `go test ./server/nativeapi -run 'TestNativeapi|OpenList'`

Expected: the package cannot compile while `native_api.go` retains conflict markers.

- [ ] **Step 3: Combine independent persisted keys**

In `consts/consts.go`, remove conflict markers and retain both groups in the same constant block:

```go
OpenListEnabledKey            = "OpenListEnabled"
OpenListBaseKey               = "OpenListBase"
OpenListUserKey               = "OpenListUser"
OpenListPassKey               = "OpenListPass"
OpenListCoverEnabledKey       = "OpenListCoverEnabled"
OpenListStreamEnabledKey      = "OpenListStreamEnabled"
LastDBAnalyzeAtKey            = "LastDBAnalyzeAt"
LastDBAnalyzeAttemptAtKey     = "LastDBAnalyzeAttemptAt"
DBAnalyzePendingKey           = "DBAnalyzePending"
DBAnalyzeFailureCountKey      = "DBAnalyzeFailureCount"
ArtConfFingerprintPropertyKey = "ArtConfFingerprint"
```

Keep the upstream comment immediately above `ArtConfFingerprintPropertyKey`.

- [ ] **Step 4: Adopt the upstream uploader type and retain the narrow bootstrap call**

In `server/nativeapi/native_api.go`, keep the upstream `artwork.Uploader` parameter and add OpenList bootstrap as the only custom constructor concern:

```go
func New(ds model.DataStore, share core.Share, playlists playlistsvc.Playlists, insights metrics.Insights, libraryService core.Library, userService core.User, maintenance core.Maintenance, pluginManager PluginManager, imgUpload artwork.Uploader) *Router {
    serveropenlist.BootstrapWithLog(ds)
    r := &Router{
        ds: ds, share: share, playlists: playlists, insights: insights,
        libs: libraryService, users: userService, maintenance: maintenance,
        pluginManager: pluginManager, imgUpload: imgUpload,
    }
    r.Handler = r.routes()
    return r
}
```

Do not move OpenList request handlers into this file; retain `api.addOpenListRoute(r)` in `routes()`.

- [ ] **Step 5: Format and rerun the focused API tests**

Run: `gofmt -w consts/consts.go server/nativeapi/native_api.go && go test ./server/nativeapi`

Expected: PASS.

### Task 2: Isolate OpenList shared-link stream redirect after authorization

**Files:**
- Modify: `server/public/handle_streams.go:20-78`
- Modify: `server/public/openlist_stream.go:1-20`
- Modify: `server/public/handle_streams_test.go:190-500`

- [ ] **Step 1: Retain the upstream invalid-share assertion**

In `server/public/handle_streams_test.go`, preserve upstream behavior for a syntactically valid token with an invalid share: status is `http.StatusBadRequest` and the streamer is not called.

```go
Expect(w.Code).To(Equal(http.StatusBadRequest))
Expect(streamer.called).To(BeFalse())
```

- [ ] **Step 2: Make OpenList redirect a dedicated helper**

Replace the exported-to-handler resolver in `server/public/openlist_stream.go` with a response-owning helper that does not redirect if OpenList cannot resolve:

```go
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
```

Add `net/http` and `log` imports to this dedicated helper file.

- [ ] **Step 3: Preserve authorization before contacting OpenList**

Delete the early `resolveOpenListSharedStreamTarget` block from `handleStream`. After `shareContainsTrack` and `HasLibraryAccess` pass, add the sole OpenList integration line:

```go
if pub.tryRedirectOpenListSharedStream(w, r, mf.ID) {
    return
}
```

This keeps invalid, expired, and unauthorized share links from triggering OpenList network requests or redirects.

- [ ] **Step 4: Make OpenList public-share tests prove success, fallback, and authorization**

Keep the existing OpenList success and fallback tests, updating them to use the new helper indirectly through `handleStream`. Add a test that configures OpenList and uses a share token that does not contain the song; assert `404`, no `Location` header, and no OpenList HTTP request (a transport counter remains zero).

```go
Expect(w.Code).To(Equal(http.StatusNotFound))
Expect(w.Header().Get("Location")).To(BeEmpty())
Expect(openListRequests).To(Equal(0))
```

- [ ] **Step 5: Run public package tests**

Run: `gofmt -w server/public/handle_streams.go server/public/openlist_stream.go server/public/handle_streams_test.go && go test ./server/public`

Expected: PASS.

### Task 3: Adopt upstream artwork API and isolate Subsonic OpenList redirects

**Files:**
- Modify: `server/subsonic/media_retrieval.go:56-104`
- Modify: `server/subsonic/openlist_cover.go:1-60`
- Modify: `server/subsonic/stream.go:19-49`
- Modify: `server/subsonic/openlist_stream.go:1-18`
- Modify: `server/subsonic/media_retrieval_test.go:1-285`
- Test: `server/subsonic/stream_test.go:15-170`

- [ ] **Step 1: Merge the test setup from both branches**

In `server/subsonic/media_retrieval_test.go`, retain upstream album/radio repositories and `lyrics.NewLyrics(ds, nil)`. Also retain OpenList environment cleanup, imports (`net/http`, `os`, `core/openlist`), and the success/fallback cover tests. The combined datastore setup must be:

```go
ds = &tests.MockDataStore{
    MockedMediaFile: mockRepo,
    MockedAlbum:     albumRepo,
    MockedRadio:     radioRepo,
}
```

- [ ] **Step 2: Use the upstream artwork return type**

Resolve `GetCoverArt` by retaining upstream retrieval and serving code:

```go
img, err := api.artwork.GetOrPlaceholder(ctx, id, size, square)
switch {
case errors.Is(err, context.Canceled):
    return nil, nil
case errors.Is(err, model.ErrNotFound):
    log.Warn(r, "Couldn't find coverArt", "id", id, err)
    return nil, newError(responses.ErrorDataNotFound, "Artwork not found")
case err != nil:
    log.Error(r, "Error retrieving coverArt", "id", id, err)
    return nil, err
}
defer img.Close()
artID, _ := model.ParseArtworkID(id)
if imghttp.WriteImageHeaders(w, r, img, artID.Hash) {
    return nil, nil
}
cnt, err := io.Copy(w, img)
if err != nil {
    log.Warn(ctx, "Error sending image", "count", cnt, err)
}
return nil, err
```

Do not retain the removed `imgReader` or `lastUpdate` variables.

- [ ] **Step 3: Make the Subsonic cover integration one delegation call**

In `server/subsonic/openlist_cover.go`, add a response-owning wrapper around the existing resolver:

```go
func (api *Router) tryRedirectOpenListCover(w http.ResponseWriter, r *http.Request, id string) bool {
    target, err := api.resolveOpenListCoverTarget(r.Context(), id)
    if err != nil || target == "" {
        return false
    }
    http.Redirect(w, r, target, http.StatusFound)
    return true
}
```

In `GetCoverArt`, call it before `GetOrPlaceholder`:

```go
if api.tryRedirectOpenListCover(w, r, id) {
    return nil, nil
}
```

The existing resolver continues to own OpenList path mapping and no OpenList protocol code is added to `media_retrieval.go`.

- [ ] **Step 4: Make the Subsonic stream integration one delegation call**

In `server/subsonic/openlist_stream.go`, replace the resolver with:

```go
func (api *Router) tryRedirectOpenListStream(w http.ResponseWriter, r *http.Request, id string) bool {
    if api == nil || api.ds == nil {
        return false
    }
    target, err := openlist.ResolveStreamRawURLBySongID(r.Context(), api.ds, id)
    if err != nil || target == "" {
        return false
    }
    http.Redirect(w, r, target, http.StatusFound)
    return true
}
```

In `Stream`, replace the inline lookup and redirect with:

```go
if api.tryRedirectOpenListStream(w, r, id) {
    return nil, nil
}
```

Keep all transcode-decision and fallback streamer code unchanged.

- [ ] **Step 5: Run Subsonic tests**

Run: `gofmt -w server/subsonic/media_retrieval.go server/subsonic/openlist_cover.go server/subsonic/stream.go server/subsonic/openlist_stream.go server/subsonic/media_retrieval_test.go && go test ./server/subsonic`

Expected: PASS.

### Task 4: Complete merge-state and UI regression verification

**Files:**
- Verify: `core/openlist/openlist.go`
- Verify: `server/openlist/*`
- Verify: `ui/src/openlist/*`
- Verify: `ui/src/audioplayer/Player.jsx`
- Verify: `ui/src/openlist/openlistStream.test.js`
- Verify: `ui/src/openlist/openlistUi.test.js`
- Verify: `ui/src/audioplayer/Player.test.jsx`

- [ ] **Step 1: Confirm the merge index is clean of conflicts**

Run: `git diff --name-only --diff-filter=U && ! rg -n '^(<<<<<<<|=======|>>>>>>>)' consts server`

Expected: no output and exit status 0.

- [ ] **Step 2: Run focused OpenList core and server regression tests**

Run: `go test ./core/openlist ./server/openlist ./server/nativeapi ./server/public ./server/subsonic`

Expected: PASS.

- [ ] **Step 3: Run OpenList UI and player regression tests**

Run: `cd ui && npm test -- --run src/openlist/openlistStream.test.js src/openlist/openlistUi.test.js src/audioplayer/Player.test.jsx`

Expected: all selected Vitest tests PASS.

- [ ] **Step 4: Verify formatting and the staged merge result**

Run: `gofmt -w consts/consts.go server/nativeapi/native_api.go server/public/handle_streams.go server/public/openlist_stream.go server/public/handle_streams_test.go server/subsonic/media_retrieval.go server/subsonic/openlist_cover.go server/subsonic/stream.go server/subsonic/openlist_stream.go server/subsonic/media_retrieval_test.go && git diff --check && git status --short`

Expected: no whitespace errors, no unmerged paths, and only the expected upstream merge plus OpenList-isolation files listed.

- [ ] **Step 5: Stage the resolved merge deliberately**

Run: `git add consts/consts.go server/nativeapi/native_api.go server/public/handle_streams.go server/public/openlist_stream.go server/public/handle_streams_test.go server/subsonic/media_retrieval.go server/subsonic/openlist_cover.go server/subsonic/stream.go server/subsonic/openlist_stream.go server/subsonic/media_retrieval_test.go docs/superpowers/specs/2026-08-19-openlist-upstream-merge-design.md docs/superpowers/plans/2026-08-19-openlist-upstream-merge.md && git status --short`

Expected: all five former conflict paths are staged without `UU`; do not create the merge commit until the full staged merge diff has been reviewed.
