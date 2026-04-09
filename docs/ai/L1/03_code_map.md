# 03 Code Map

> Directory map and the fastest way to find the code that implements each behavior.

## Top-Level Directories

| Path | Responsibility |
| --- | --- |
| `connection-setup/` | mock REST session start/stop API |
| `websocket-receive-audio/` | mock WebSocket audio receiver |
| `go-publish-video/` | actual Agora publisher sample |
| `docs/ai/` | progressive-disclosure docs |

## Quick Navigation

- session setup bug:
  - start in `connection-setup/session_test_receiver.py`
- WebSocket auth or file output issue:
  - start in `websocket-receive-audio/websocket_test_receiver.py`
- publish or SDK issue:
  - start in `go-publish-video/child.go`
- sample looping or timing issue:
  - start in `go-publish-video/parent.go`

## Go Publisher Files

| File | Responsibility |
| --- | --- |
| `go-publish-video/parent.go` | parent process, media file loop, IPC send |
| `go-publish-video/child.go` | child process, Agora SDK init/publish |
| `go-publish-video/child_ipc.go` | framed status/log responses |
| `go-publish-video/subscriber.go` | basic subscriber utility |
| `go-publish-video/ipc/ipc_defs.fbs` | IPC schema |
| `go-publish-video/Makefile` | build and test entrypoints |
| `go-publish-video/test_data/` | sample audio/video assets |

## Go Publisher Supporting Files

| File | Responsibility |
| --- | --- |
| `go-publish-video/go.mod` | module deps and local SDK override |
| `go-publish-video/go.sum` | dependency lock |
| `go-publish-video/test_e2e.sh` | shell entrypoint for manual e2e checks |
| `go-publish-video/child_test.go` | child-side focused tests |
| `go-publish-video/parent_test.go` | parent loop and conversion tests |
| `go-publish-video/ipc_test.go` | FlatBuffers message coverage |

## Python Files

| File | Responsibility |
| --- | --- |
| `connection-setup/session_test_receiver.py` | mock session receiver |
| `connection-setup/session_start.py` | client for start request |
| `connection-setup/session_stop.py` | client for stop request |
| `websocket-receive-audio/websocket_test_receiver.py` | mock WS receiver |
| `websocket-receive-audio/websocket_audio_sender.py` | WS sender utility |

## Test Files

| File | Responsibility |
| --- | --- |
| `connection-setup/test_session_receiver.py` | receiver behavior tests |
| `connection-setup/test_session_start.py` | start-client tests |
| `connection-setup/test_session_stop.py` | stop-client tests |
| `websocket-receive-audio/test_websocket_receiver.py` | receiver protocol tests |
| `websocket-receive-audio/test_websocket_sender.py` | sender utility tests |

## Where To Edit

- change media publish format:
  - `go-publish-video/parent.go`
  - `go-publish-video/child.go`
- change Agora SDK/version instructions:
  - `go-publish-video/README.md`
- change REST or WS mock protocol:
  - respective Python README and receiver file

## High-Signal Grep Targets

- `enableStringUID`
- `SendVideoFrame`
- `SendAudioPcmData`
- `session/start`
- `voice_end`
- `heartbeat`

## Generated Or Runtime Artifacts

- `go-publish-video/parent` and `go-publish-video/child` are built binaries
- `go-publish-video/agora_child_sdk.log` and `go-publish-video/agoraapi.log` are runtime logs
- `__pycache__/` and `.pytest_cache/` are generated artifacts, not source of truth

## Change Boundaries

- keep protocol examples in Python simple and standalone
- avoid coupling mock receivers directly to the Go publisher
- keep FlatBuffers schema changes coordinated with both parent and child code

## Related Deep Dives

- [Go Publisher Pipeline](deep_dives/go_publisher_pipeline.md) — exact parent/child handoff and track ownership
- [ConvoAI Protocol Notes](deep_dives/convoai_protocol.md) — message families and repo boundaries
