# ConvoAI to Video — Integration Guide

This guide walks avatar providers through integrating with Agora's ConvoAI platform. By the end you will have a working pipeline that receives AI agent speech, generates video, and publishes both audio and video back into an Agora channel.

## Architecture Overview

```
Agora ConvoAI Platform
        │
        ├─── POST /session/start ──► Your REST API (returns WebSocket address + token)
        │
        ├─── WebSocket audio stream ──► Your WebSocket server
        │                                    │
        │                                    ▼
        │                              Your video generation pipeline
        │                                    │
        │                                    ▼
        └─── Agora channel ◄────── Go publisher (audio + video)
```

1. **Agora calls your REST API** with session parameters (avatar, quality, Agora channel credentials).
2. **You return** a WebSocket address and session token.
3. **Agora streams audio** to your WebSocket — this is the AI agent's speech.
4. **You generate video** from the audio (e.g. lip-synced avatar).
5. **You publish audio + video** back into the Agora channel using the Go publisher.

## Components

### 1. Connection Setup API — *you implement this*

REST endpoints that Agora calls to establish and tear down sessions.

| Endpoint | Purpose |
|----------|---------|
| `POST /session/start` | Receive session config, return WebSocket address + session token |
| `DELETE /session/stop` | Tear down session and release resources |

Full specification: [connection-setup/README.md](../../connection-setup/README.md)

**Key fields in the start request:**
- `avatar_id`, `quality`, `video_encoding` — avatar and stream settings
- `agora_settings` — App ID, token, channel, UID, enable_string_uid
- `activity_idle_timeout` — auto-terminate after idle period

**Response must include:**
- `session_id` — for session management
- `websocket_address` — where Agora will stream audio
- `session_token` — for WebSocket authentication

### 2. WebSocket Audio Receiver — *you implement this*

WebSocket server that receives real-time audio from ConvoAI.

Full specification: [websocket-receive-audio/README.md](../../websocket-receive-audio/README.md)

**Message flow:**
1. `init` — session configuration (avatar, quality, codec, Agora credentials)
2. `voice` — base64-encoded audio chunks (PCM16/PCM8/OPUS)
3. `voice_end` — marks end of a speech segment
4. `voice_interrupt` — immediately stop current speech
5. `heartbeat` — keep-alive every 10 seconds
6. `special` — reserved for gestures, tool calls, etc.

Authentication: Bearer token in WebSocket connection headers.

### 3. Go Audio/Video Publisher — *use as-is or adapt*

Publishes YUV video frames and PCM audio into an Agora channel using the Agora Golang Server SDK.

Full setup and usage: [go-publish-video/README.md](../../go-publish-video/README.md)

**Quick start:**
```bash
cd go-publish-video
make
./parent -appID "<your_app_id>" -channelName "<channel>" -userID "<uid>"
```

**Supported codecs:** H264, VP8, AV1

**How to feed your own media:**
Modify `parent.go` to send your generated YUV video frames and PCM audio instead of the bundled test data. The child process handles all Agora SDK interaction via IPC.

## Integration Checklist

1. Implement `POST /session/start` and `DELETE /session/stop`
2. Implement WebSocket server that accepts audio streaming commands
3. Build your video generation pipeline (audio → avatar animation → YUV frames)
4. Feed generated frames + received audio into the Go publisher
5. Test end-to-end with Agora Web Demo: join the same channel at https://webdemo.agora.io/basicVideoCall/index.html

## Testing

Each component has unit tests that run without Agora credentials:

```bash
# Go publisher — IPC, message framing, codec config
cd go-publish-video && make test

# Connection setup — session validation, mock server
cd connection-setup && pip install -r requirements.txt -r requirements-dev.txt && pytest -v

# WebSocket — token validation, audio handling, message protocol
cd websocket-receive-audio && pip install -r requirements.txt -r requirements-dev.txt && pytest -v
```

E2E tests (requires Agora App ID and App Certificate):
```bash
cd go-publish-video
APP_ID=<your_app_id> APP_CERT=<your_app_cert> make test-e2e
```

## Technical Notes

- **IPC**: Parent/child communicate via FlatBuffers over stdin/stdout
- **String vs numeric UIDs**: Controlled by `enable_string_uid` in the init message; the Go publisher must match
- **AV1 codec**: Requires Go SDK >= v2.4.10 and native SDK >= v4.4.32.164. Older SDKs silently fall back to H264.
- **Tokens**: Generated externally and passed to the publisher via the `-token` flag. The `cmd/tokengen/` tool is provided for testing only.
