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

## Important Boundary

- these Python components validate protocol shape
- they do not implement the actual avatar engine
- they do not automatically pipe media into `go-publish-video/`

## Production Adaptation

- replace the in-memory stores with real session management
- replace the WAV-save behavior with avatar processing
- keep auth and token validation stronger than the mock examples
