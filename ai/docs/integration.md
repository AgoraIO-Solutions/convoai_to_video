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
- `area` — geographic hint for server selection (default: `GLOBAL`)
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
2. `voice` — base64-encoded audio chunks (PCM16/PCM8/OPUS), carrying a `segment_id`
3. `voice_end` — ends that segment, carrying the same `segment_id`
4. `voice_interrupt` — stop and discard everything buffered and playing, never part of it
5. `heartbeat` — keep-alive every 10 seconds
6. `special` — gestures and other instructions; one bound to speech carries that
   segment's `segment_id` and is acted on when the segment begins playing

Segments are queued and played in arrival order. A `voice` frame arriving while an
earlier segment is playing is appended, not a signal to replace what is playing.

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

No unit-test suite is included. Use the component READMEs for
[REST session checks](../../connection-setup/README.md#testing) and the
[manual WebSocket flow](../../websocket-receive-audio/README.md#testing).
The REST clients exit non-zero if any check fails. WebSocket sender success
confirms transmission; inspect the receiver's WAV to verify the output.

The Go audio smoke check requires the native SDK, an Agora App ID, and App Certificate:
```bash
cd go-publish-video
APP_ID=<your_app_id> APP_CERT=<your_app_cert> make test-e2e
```

It verifies at least 10 non-empty audio frames from the expected publisher
in string and numeric UID modes. Verify video and A/V synchronization manually.

## Technical Notes

- **IPC**: Parent/child communicate via FlatBuffers over stdin/stdout
- **String vs numeric UIDs**: Controlled by `enable_string_uid` in the init message; the Go publisher must match
- **AV1 codec**: has minimum SDK versions — see `go-publish-video/README.md`. Older SDKs silently fall back to H264.
- **Tokens**: Generated externally and passed to the publisher via the `-token` flag. The `cmd/tokengen/` tool is provided for testing only.
