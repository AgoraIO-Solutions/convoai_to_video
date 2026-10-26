# ConvoAI Protocol Notes

## Purpose

Capture what the Python parts of this repo model and what they intentionally do not model.

## Session Setup Side

- `connection-setup/` provides mock request handlers for session start/stop
- the start path returns:
  - session id
  - WebSocket address
  - session token
- the stop path tears down the mock in-memory session

## WebSocket Side

- `websocket-receive-audio/` models the inbound audio stream from Agora ConvoAI
- expected message families:
  - `init`
  - `voice`
  - `voice_end`
  - `voice_interrupt`
  - `heartbeat`
  - `special` (including avatar actions)

`voice` and `voice_end` carry a `segment_id` naming the segment they belong to.
Segments are queued and played in arrival order: a `voice` frame arriving during
playback is appended, never a signal to stop or replace. Playback stops only on
`voice_interrupt`, which discards everything buffered and playing.

Avatar actions use `special` with a JSON string in `content`, for example
`{"type":"action","action":"wave"}`. An action carrying a `segment_id` is acted on
when that segment begins playing, because the segment may be queued behind other
speech; one without it is acted on immediately. Inline LLM text and developer
`speak`/RTM text produce an action bound to their segment; the built-in
`perform_gesture` tool is a real-time instruction belonging to no segment, so it
carries no id. Configuration and examples are in
`websocket-receive-audio/README.md`.

## Important Boundary

- these Python components validate protocol shape
- they do not implement the actual avatar engine
- the reference receiver does not implement avatar action handling
- they do not automatically pipe media into `go-publish-video/`

## Production Adaptation

- replace the in-memory stores with real session management
- replace the WAV-save behavior with avatar processing
- keep auth and token validation stronger than the mock examples
