# 06 Interfaces

> External contracts, message shapes, and repo boundaries.

## REST Session Setup

- endpoint family lives in `connection-setup/`
- purpose:
  - start session
  - stop session
- expected output from start:
  - session id
  - WebSocket address
  - session token

## Session Start Inputs

- avatar or scene identity
- quality or rendering preferences
- Agora channel credentials
- UID mode expectations when applicable

## Session Stop Inputs

- session id
- auth context used by the provider implementation

## REST Ownership

- Agora is the caller
- avatar provider owns request validation and lifecycle
- this repo only shows mock request/response handling

## WebSocket Audio Flow

- endpoint family lives in `websocket-receive-audio/`
- expected message types include:
  - init
  - voice
  - voice_end
  - voice_interrupt
  - heartbeat

## WebSocket Receiver Expectations

- bearer token validation happens before accepting media
- `init` must arrive before voice frames
- audio chunks are appended until `voice_end`
- `voice_interrupt` can abort the current utterance
- heartbeats keep the session alive

## Output Behavior In The Mock Receiver

- voice data is saved to WAV files
- sessions are tracked in memory
- mock implementation is optimized for protocol validation, not scale

## Publisher IPC

- schema file:
  - `go-publish-video/ipc/ipc_defs.fbs`
- command types:
  - init
  - write video sample
  - write audio sample
  - close
- response types:
  - status
  - log

## IPC Ownership

- parent writes framed IPC messages to child stdin
- child reads messages and publishes them
- child writes structured status responses over stdout
- human-readable logs go to stderr

## IPC Payload Expectations

- video sample must include:
  - width
  - height
  - pixel buffer
  - timestamp
- audio sample must include:
  - sample rate
  - channel count
  - PCM bytes
  - timestamp

## Media Contracts

- audio frames:
  - PCM16
  - default 16 kHz mono
- video frames on workaround branch:
  - input file format: `RGB24`
  - sent to SDK as `RGBA`

## UID / Token Contract

- all channel participants must use the same UID mode
- token generation must match UID mode
- numeric UID mode is the simplest path for browser verification

## Out-Of-Scope In This Repo

- avatar generation internals
- production auth storage
- fleet management for multiple concurrent publishers
- browser client implementation details

## Related Deep Dives

- [ConvoAI Protocol Notes](deep_dives/convoai_protocol.md) — session and WebSocket mock contracts
- [Go Publisher Pipeline](deep_dives/go_publisher_pipeline.md) — IPC sample details and timing
