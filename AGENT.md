# Agent Instructions

## Project

Agora ConvoAI to Video — enables avatar providers to receive AI agent speech, generate video, and publish audio/video back into Agora channels.

## Documentation

Detailed integration guide and architecture: [ai/docs/integration.md](ai/docs/integration.md)

Component-specific docs:
- [connection-setup/README.md](connection-setup/README.md) — REST API specification
- [websocket-receive-audio/README.md](websocket-receive-audio/README.md) — WebSocket protocol specification
- [go-publish-video/README.md](go-publish-video/README.md) — Go publisher setup, build, and usage

## Repository Structure

```
convoai_to_video/
├── connection-setup/          # REST API (Python) — session start/stop
├── websocket-receive-audio/   # WebSocket server (Python) — audio streaming
├── go-publish-video/          # Agora publisher (Go + CGO)
│   ├── parent.go              # Orchestrator (pure Go, no CGO)
│   ├── child.go               # Agora SDK worker (CGO)
│   ├── child_ipc.go           # IPC message handling
│   ├── subscriber.go          # E2E test subscriber
│   ├── cmd/tokengen/          # Standalone token generator (separate module)
│   ├── test_e2e.sh            # E2E test orchestration
│   └── Makefile               # Build targets
├── ai/docs/                   # Integration documentation
└── sequence-diagram.svg       # Architecture flow diagram
```

## Build and Test

```bash
# Go publisher
cd go-publish-video && make        # build
cd go-publish-video && make test   # unit tests

# Python components
cd connection-setup && pytest -v
cd websocket-receive-audio && pytest -v

# E2E (requires Agora credentials)
cd go-publish-video
APP_ID=<your_app_id> APP_CERT=<your_app_cert> make test-e2e
```

## Key Conventions

- **No secrets in code**: App IDs, certificates, and tokens are always passed via flags or environment variables, never hardcoded
- **Token generation**: Lives only in test tooling (`cmd/tokengen/`), not in the main publisher code. Tokens are passed to the publisher via `-token` flag by the ConvoAI platform.
- **go.mod replace directive**: Must point to the local Agora Golang Server SDK checkout. The path differs between Linux deploy (`/home/ubuntu/`) and local Mac dev.
- **Native SDK**: Downloaded via `bash scripts/install_agora_sdk.sh` from the Agora Golang Server SDK directory
- **AV1 codec**: Requires Go SDK >= v2.4.10 and native SDK >= v4.4.32.164. Older versions silently fall back to H264.
- **IPC**: Parent/child processes communicate via FlatBuffers over stdin/stdout. Child logs go to stderr with `[agora_worker]` prefix.
- **String vs numeric UIDs**: Controlled by `-enableStringUID` flag; must match between publisher and channel participants
