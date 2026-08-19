# OpenList Feishin Transcode Streaming Design

## Goal

Allow Feishin's OpenSubsonic playback path to play OpenList-backed tracks while
preserving the endpoint's existing authentication and token validation.

## Cause and boundary

The legacy Subsonic `stream` handler redirects eligible tracks through
`tryRedirectOpenListStream`. Feishin uses the OpenSubsonic
`getTranscodeStream` handler instead. That handler validates its request and
then unconditionally creates a local Navidrome stream, which fails for media
available only through OpenList.

## Design

After `GetTranscodeStream` has validated `mediaId`, `mediaType`, and
`transcodeParams`, and has resolved the token to a stream request, call the
existing `tryRedirectOpenListStream` helper with `mediaID`. If it resolves an
OpenList raw URL, return its 302 response. If resolution is unavailable or
fails, continue unchanged to `NewStream`.

The redirect is deliberately after token validation. This keeps OpenList from
becoming a way to bypass the OpenSubsonic playback capability and avoids an
OpenList request for invalid or stale tokens.

## Testing

Add focused unit coverage alongside the transcode handler proving that a
validated OpenList-backed request redirects without calling the local streamer.
The existing tests continue to cover invalid/stale tokens and local-stream
behavior, protecting the unchanged fallback path.
