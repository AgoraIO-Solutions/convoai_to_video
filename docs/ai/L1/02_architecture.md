# 02 Architecture

> High-level system shape, data flow, and how the three repo components fit together.

## Main Flow

1. Agora calls your session setup endpoint
2. Your service returns a WebSocket URL and token
3. Agora streams agent speech to your WebSocket endpoint
4. Your avatar system generates video frames from that speech
5. The Go publisher sends audio and video into the Agora channel

## System Boundaries

- Agora owns:
  - channel, token validation, subscriber delivery
  - the ConvoAI side that calls the avatar provider session API
- avatar provider owns:
  - session setup endpoint
  - WebSocket receiver
  - avatar generation pipeline
  - final publish process
- this repo provides:
  - protocol examples for session setup and audio ingest
  - a reference Go publisher implementation

## Components

- `connection-setup/`
  - mock REST API for `session/start` and `session/stop`
- `websocket-receive-audio/`
  - mock WebSocket receiver for audio chunks and control messages
- `go-publish-video/`
  - `parent.go` reads local media and sends framed IPC messages
  - `child.go` owns the Agora SDK objects and publishes tracks
  - `child_ipc.go` serializes status/log responses back to parent

## Reference vs Production

| Area | In This Repo | Production Expectation |
| --- | --- | --- |
| Session REST | mocked receiver and helper clients | real service owned by avatar provider |
| Audio WebSocket | mocked receiver and sender utility | real streaming ingest endpoint |
| Video generation | not implemented | real avatar engine |
| Agora publish | working reference sample | can be adapted or embedded |

## Publisher Shape

- parent process:
  - pure Go
  - reads PCM audio and raw video frames from disk
  - packages samples with FlatBuffers over stdin/stdout IPC
- child process:
  - CGO / Agora SDK
  - initializes SDK
  - creates audio/video tracks
  - publishes frames into Agora

## Parent/Child Split

- parent responsibilities:
  - open media files
  - pace the loop
  - convert raw sample formats
  - recover from EOF by restarting the sample loop
- child responsibilities:
  - create Agora service and connection
  - configure encoder settings
  - own local audio and video tracks
  - send status back to parent

## Current Media Path

1. parent reads PCM audio chunks
2. parent reads `RGB24` video frames
3. parent packs `RGB24` into `RGBA`
4. parent sends media samples over FlatBuffers IPC
5. child publishes PCM audio and raw `RGBA` video through Agora

## Testing Shape

- Python components are unit-testable without Agora
- Go IPC/tests are unit-testable without Agora
- full end-to-end validation requires:
  - valid credentials
  - a live channel
  - a viewer or subscriber

## Architectural Constraints

- Python mocks do not automatically feed the Go publisher
- all participants in a channel must use the same UID mode
- SDK ownership stays in the child process
- parent should remain free of CGO/Agora dependencies

## Current Workaround Path

- native SDK line: `4.4.31`
- raw input: `RGB24`
- publish format: `RGBA`
- branch target: customer workaround for Modal/gVisor compatibility

## Related Deep Dives

- [Go Publisher Pipeline](deep_dives/go_publisher_pipeline.md) — IPC framing, timestamps, and media ownership
- [ConvoAI Protocol Notes](deep_dives/convoai_protocol.md) — session and WebSocket protocol boundaries
