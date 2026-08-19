# OpenList upstream merge design

## Goal

Complete the in-progress merge from `upstream/master` while preserving the
OpenList settings, cover-art, and stream-resolution features. Keep the custom
integration isolated so future upstream merges touch only small, explicit
extension points.

## Scope

- Resolve the four current conflicts by retaining upstream behavior and adding
  the OpenList extension beside it.
- Keep OpenList code in its dedicated packages and files:
  `core/openlist`, `server/openlist`, `server/nativeapi/openlist.go`,
  `server/public/openlist_stream.go`, `server/subsonic/openlist_*.go`, and
  `ui/src/openlist`.
- Restrict edits to upstream-owned shared files to narrow registration or
  delegation points.
- Add or retain focused tests for configuration, stream resolution, public
  sharing, Subsonic streaming, cover art, and UI registration.

## Conflict resolution approach

1. In shared constants, retain both upstream constants and the OpenList
   property keys; neither set changes the meaning of the other.
2. In native API routing, retain upstream route setup and register the
   dedicated `addOpenListRoute` helper from the existing router bootstrap.
3. In public and Subsonic streaming paths, retain upstream request and
   streaming logic. Delegate OpenList-specific resolution through the existing
   dedicated helper files, falling back to the upstream streamer whenever
   OpenList is disabled, unsupported, or cannot resolve a raw URL.
4. Do not copy OpenList protocol, authentication, or configuration code into
   upstream-owned handlers. Those concerns remain in `core/openlist` and
   `server/openlist`.

## Compatibility contract

- Administrators can read and update OpenList configuration through the native
  API and UI without exposing stored passwords.
- When enabled, OpenList can provide artwork and a raw URL for eligible media.
- If OpenList is disabled or resolution fails, Navidrome keeps the normal
  upstream cover and streaming behavior.
- Existing non-OpenList deployments behave as upstream does.

## Verification

- Confirm there are no unresolved paths or conflict markers.
- Run focused Go tests for `core/openlist`, `server/openlist`,
  `server/nativeapi`, `server/public`, and `server/subsonic`.
- Run the focused UI OpenList and player tests.
- Run formatting and static checks for modified files where available.

## Future-merge guardrails

- Treat dedicated OpenList files as the extension boundary.
- Keep shared-file changes limited to a constant declaration, route
  registration, or a single delegation call.
- Test fallback behavior at every shared stream boundary so upstream changes
  cannot silently bypass the normal Navidrome path.
