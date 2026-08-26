# OpenList merge-isolation design

## Goal

Further reduce the chance that upstream `master` merges conflict with the
OpenList integration. Keep only the shared production lines that are required
to preserve Navidrome authorization, timeout, and routing behavior.

## Decision

Move OpenList's persisted-property key names out of `consts/consts.go` and
into `core/openlist`. Move OpenList-specific public-share and Subsonic-cover
tests out of the large upstream-owned handler test files into dedicated
OpenList test files. Retain the existing narrow production seams.

## Why this boundary

Three shared seams are intentionally retained:

- Native API construction boots the persisted OpenList state and routes call
  the dedicated OpenList registration helper.
- The public share-stream handler delegates only after share membership and
  library authorization succeed.
- The Subsonic cover and stream handlers delegate after their existing request
  setup, including the cover-art deadline.

These calls must remain in the upstream handlers because moving them earlier
would weaken authorization or timeout behavior, while replacing them with a
generic hook would expand the upstream-facing API surface. Each is one
delegation call and should be reviewed when upstream changes its handler.

## Changes

### Persisted keys

`core/openlist` exports the six OpenList property keys it owns. The OpenList
configuration implementation and its tests use those names. No OpenList key
remains in the shared `consts` block, leaving that file wholly upstream-owned.
The persisted string values do not change, so existing configurations remain
compatible.

### Test ownership

Move the `handleStream OpenList` suite from `handle_streams_test.go` to
`openlist_stream_test.go` in `server/public`. Move OpenList cover redirect,
fallback, deadline, and resolver-seam tests from `media_retrieval_test.go` to
`openlist_cover_test.go` in `server/subsonic`.

The extracted suites keep their own test doubles and setup. They must restore
all OpenList environment variables, reset OpenList global state during cleanup,
and restore any injected HTTP client. Upstream handler tests retain only
upstream behavior.

## Compatibility and error handling

- Property names and values are unchanged.
- OpenList remains optional: resolver errors or empty URLs fall back to the
  upstream cover/stream behavior.
- Public share authorization continues to happen before OpenList network
  access.
- OpenList cover resolution continues to use the existing ten-second artwork
  context.

## Verification

- `go test ./core/openlist ./server/nativeapi ./server/public ./server/subsonic`
- Confirm `consts/consts.go` has no OpenList symbols.
- Confirm the three production delegation seams remain singular and tests
  cover success, fallback, authorization, and deadline propagation.
- `git diff --check`.

## Non-goals

- Do not replace the shared handler seams with a plugin framework or global
  middleware.
- Do not alter the OpenList API, UI, persisted property values, authentication,
  or redirect semantics.
