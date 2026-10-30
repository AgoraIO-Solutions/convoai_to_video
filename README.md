# Agora ConvoAI Agent Speech to Video Integration

This repository provides a protocol and reference implementation for **avatar providers** to integrate with Agora's ConvoAI platform. It enables receiving an AI agent's speech output, generating video (e.g. an animated avatar), and publishing both the audio and video back into an Agora channel for low-latency, global distribution.

## How It Works

1. **Agora calls your REST API** with session parameters (avatar ID, quality, area, Agora channel credentials)
2. **You return a WebSocket address** and session token
3. **Agora streams audio** to your WebSocket — this is the AI agent's speech
4. **You generate video** from the audio (e.g. lip-synced avatar animation)
5. **You publish audio + video** back into the Agora channel using the Go publisher

## Architecture Flow

```mermaid
sequenceDiagram
    participant C as Agora ConvoAI
    participant R as Your REST API
    participant W as Your WebSocket
    participant A as Agora Channel

    rect rgb(245,245,245)
    note over C,R: 1. Session setup
    C->>R: POST /session/start<br/>{avatar_id, quality, video_encoding,<br/>activity_idle_timeout, area,<br/>agora_agent_id, agora_settings}
    R-->>C: {session_id, websocket_address, session_token}
    end

    rect rgb(245,245,245)
    note over C,W: 2. WebSocket connect
    C->>W: Connect, Authorization: Bearer {session_token}
    C->>W: {command:"init", session_id, avatar_id, ...,<br/>agora_agent_id, agora_settings}
    end

    rect rgb(245,245,245)
    note over W,A: 3. Join and publish
    W->>A: Connect (Go SDK), then PushVideoFrame (I420)<br/>+ PushAudioPcmData (PCM)
    end

    rect rgb(245,245,245)
    note over C,W: 4. Speech, as queued segments
    loop one segment per utterance
        C->>W: {command:"voice", audio, sampleRate,<br/>encoding, segment_id, event_id}
        opt action bound to this segment
            C->>W: {command:"special", content, segment_id, event_id}
        end
        C->>W: {command:"voice_end", segment_id, event_id}
    end
    note over W: queue segments, play in arrival order
    opt cancel
        C->>W: {command:"voice_interrupt", event_id}
        note over W: discard everything buffered and playing
    end
    opt every 10s
        C->>W: {command:"heartbeat", event_id, timestamp}
        W-->>C: {command:"heartbeat_ack", ...}
    end
    end

    rect rgb(245,245,245)
    note over C,R: 5. Teardown
    C->>R: DELETE /session/stop {session_id, session_token}
    end
```

## Implementation Components

### 1. Connection Setup API — *you implement this*
[connection-setup/](./connection-setup/) — REST endpoints (`POST /session/start`, `DELETE /session/stop`) that Agora calls to establish and tear down sessions. You return a WebSocket address and session token.

### 2. WebSocket Audio Streaming — *you implement this*
[websocket-receive-audio/](./websocket-receive-audio/) — WebSocket server that receives real-time audio from ConvoAI. You process the audio through your video generation pipeline.

### 3. Go Audio/Video Publishing — *use as-is or adapt*
[go-publish-video/](./go-publish-video/) — Publishes YUV video frames and PCM audio into an Agora channel. Feed your generated video and the received audio into this component to push the stream back to users.

## Driving the Avatar: Actions, Gestures and Cues

Beyond audio, the agent can tell your avatar to *do* something — wave, smile, hold a mood for a
few seconds. These all travel on the existing `special` command, so there is no second channel
to implement and no new connection:

```json
{
  "command": "special",
  "content": "{\"type\":\"action\",\"action\":\"happy\",\"params\":{\"in_seconds\":3}}",
  "segment_id": "seg-7f3a1c",
  "event_id": "cd3c81e682af4bd5a63d60be07b0b537"
}
```

**The names and their meaning are yours.** A one-shot gesture and a cue that holds a style until
the end of a response are both just names here — the protocol carries the trigger and the
binding to a segment, not the meaning. You declare which names you accept, and which parameters
each takes; nothing outside that is ever sent.

The same command carries two other things, both optional and both off unless the caller turns
them on: the **spoken transcript**, delivered ahead of the audio it describes, and the
**conversation state** — listening, thinking, speaking — which tells the avatar it is about to
have to speak roughly a second before any audio arrives.

Three things are worth reading before you implement any of it:

- **the ignore rule** — what to do with a `type`, a key or an action name you do not recognise.
  A receiver that errors or disconnects on an unknown field will break the next time one is
  added.
- **`segment_id`** — present when the message belongs to a specific piece of speech, absent when
  it is immediate. The difference is what lets you hold a gesture with its audio instead of
  playing it against whatever is on screen.
- **what `voice_end` does and does not mean.** Audio is sent faster than it plays — up to 17x —
  so `voice_end` can close a segment tens of seconds before the avatar stops speaking. It means
  no more audio is coming, not that speech has finished.

→ **[Special Command and Avatar Actions](./websocket-receive-audio/README.md#6-special-command)**
covers the frame, both ways an action is triggered with a worked example of each, and the
configuration that drives them.

## Testing

No unit-test suite is included. The Python scripts provide manual checks;
the Go publisher has a credentialed audio smoke check. `make test` exits
non-zero with an explanation because there is no unit-test suite to run.

### REST session checks

Follow [connection setup testing](connection-setup/README.md#testing) to run
the mock server and the session start/stop scripts. Each client exits non-zero
if any check fails.

### Python — the mock pair
```bash
cd websocket-receive-audio
pip install -r requirements.txt
python websocket_test_receiver.py       # terminal 1
python websocket_audio_sender.py        # terminal 2
```
The receiver writes `received_audio_<timestamp>.wav`; compare it with
`input.wav` — same sample rate, same length. `connection-setup/` has the same
shape for the REST side.

The sender exits non-zero on a send or listener error. A successful exit
confirms transmission only; inspect the receiver's WAV to verify the output.

### Agora audio smoke check (requires SDK and credentials)

```bash
cd go-publish-video
APP_ID=<your_app_id> APP_CERT=<your_app_cert> make test-e2e
```

This checks that a subscriber receives at least 10 non-empty audio frames
from the expected publisher in both UID modes. Video and A/V synchronization
need the manual viewer check below.

### End-to-end (requires Agora credentials)
```bash
cd go-publish-video
make
./parent -appID "<your_app_id>" -channelName "<your_channel>" -userID "<your_uid>"
```

### Viewing the stream

Open [Agora Web Demo](https://webdemo.agora.io/basicVideoCall/index.html) and join with:
- **App ID**: same App ID used by the publisher
- **Channel**: same channel name
- **Token**: if your app has certificates enabled, generate a subscriber token using `go-publish-video/cmd/tokengen/` (see [go-publish-video/README.md](go-publish-video/README.md#manual-testing))
- **UID**: any UID that is not the publisher's UID

The web demo uses **numeric UIDs**. If your publisher uses numeric UIDs (`-enableStringUID=false`), the web viewer will see the stream directly. If your publisher uses string UIDs (`-enableStringUID=true`), the web demo cannot join the same session — use the Agora Video SDK sample app or another string-UID-capable client instead.

## String vs Numeric UIDs

Agora channels support two UID modes. All participants in a channel **must use the same mode**.

| Mode | Flag | UID format | Web demo compatible |
|------|------|-----------|---------------------|
| Numeric | `-enableStringUID=false` | Integers as strings, e.g. `"100"` | Yes |
| String | `-enableStringUID=true` (default) | Any string, e.g. `"avatar_agent"` | No |

- The ConvoAI platform passes `enable_string_uid` in the session start request — your publisher must match this setting
- The `cmd/tokengen/` tool generates the correct token type based on `-enableStringUID`
- When in doubt, use numeric UIDs (`-enableStringUID=false`) for easier testing with the web demo

## Video Codecs

The Go publisher supports H264, VP8, and AV1 via the `-videoCodec` flag.

- **H264** (default): Widest device and browser compatibility
- **VP8**: Open source, good web compatibility
- **AV1**: Best compression efficiency, but **requires your Agora App ID to have AV1 enabled on the Agora backend** — contact Agora support to enable it. Without backend enablement, the stream will silently fall back to H264. AV1 also has minimum SDK versions — see `go-publish-video/README.md`, which owns them.

## Use Cases

**Interactive AI Avatars**
- Brand mascots and talking cartoon characters for entertainment and marketing
- Virtual assistants and customer service representatives with lifelike appearance

**Dynamic Content Creation**
- Interactive movies and choose-your-own-adventure experiences that adapt to viewer input
- Automated video hosts for news, podcasts, and live streaming

**Real-time Visualization**
- Architectural design consultations with live 3D building and interior visualizations
- Scientific simulations showing molecular interactions, physics concepts, and biological processes

**Educational & Training Applications**
- AI tutors with visual demonstrations for personalized education
- Medical procedure training with anatomical models and surgical simulations

**Creative & Analytical Presentation**
- Data visualization for financial analysis, weather forecasting, and business intelligence
- Virtual real estate tours and travel experiences with immersive environments
