# OpenList Feishin Transcode Streaming Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let Feishin play OpenList-backed tracks through OpenSubsonic `getTranscodeStream`.

**Architecture:** Reuse the existing Subsonic OpenList redirect helper only after `getTranscodeStream` has completed its established request and token validation. Resolution failures preserve the existing local-stream fallback.

**Tech Stack:** Go, Ginkgo/Gomega, Go `net/http/httptest`.

---

### Task 1: Redirect validated OpenSubsonic transcode requests through OpenList

**Files:**
- Modify: `server/subsonic/transcode.go:364-449`
- Test: `server/subsonic/transcode_test.go:422-515`

- [ ] **Step 1: Write the failing regression test**

In the `GetTranscodeStream` test suite, configure the mock media repository
with an OpenList-backed song, configure a test OpenList server to return a raw
URL, set `mockTD.resolvedReq`, and call:

```go
r := newGetRequest("mediaId=song-1", "mediaType=song", "transcodeParams=valid-token")
resp, err := router.GetTranscodeStream(w, r)
Expect(err).ToNot(HaveOccurred())
Expect(resp).To(BeNil())
Expect(w.Code).To(Equal(http.StatusFound))
Expect(w.Header().Get("Location")).To(Equal(srv.URL + "/d/Artist/Album/track.flac"))
Expect(fakeStreamer.captured).To(BeNil())
```

- [ ] **Step 2: Run the focused test and confirm the pre-fix failure**

Run: `go test ./server/subsonic -run TestSubsonic -ginkgo.focus='redirects validated transcode streams to OpenList'`

Expected: FAIL because `GetTranscodeStream` invokes `NewStream` instead of returning the OpenList redirect.

- [ ] **Step 3: Add the minimal redirect seam**

Immediately after `ResolveRequestFromToken` succeeds in
`GetTranscodeStream`, add:

```go
if api.tryRedirectOpenListStream(w, r, mediaID) {

    return nil, nil
}
```

Keep it after token validation and before `api.streamer.NewStream`.

- [ ] **Step 4: Re-run focused and package tests**

Run: `go test ./server/subsonic -run TestSubsonic -ginkgo.focus='redirects validated transcode streams to OpenList' && go test ./server/subsonic`

Expected: PASS. The focused test observes the 302 redirect and the package suite confirms existing token-error and streamer paths are unchanged.

- [ ] **Step 5: Commit**

```bash
git add server/subsonic/transcode.go server/subsonic/transcode_test.go docs/superpowers/specs/2026-08-19-openlist-feishin-transcode-design.md docs/superpowers/plans/2026-08-19-openlist-feishin-transcode.md
git commit -m "fix(openlist): support OpenSubsonic transcode streams"
```
