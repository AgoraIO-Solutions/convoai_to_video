# WebSocket API Documentation

This document describes the WebSocket API for driving remote audio and video generation sessions with streaming audio.

## Connection Setup

### Headers

Required headers for establishing the WebSocket connection:

```python
headers = {
    "authorization": "Bearer {session_token}"
}

# Use with websockets library:
websocket = await websockets.connect(
    "ws://localhost:8765", 
    additional_headers=headers
)
```

#### Header Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| authorization | string | Yes | Bearer token authentication. Format: `Bearer {session_token}` where `{session_token}` is the token obtained from the initial connection setup endpoint. |

## Message Protocol

All messages are sent as JSON strings over the WebSocket connection. The API uses a command-based protocol where each message contains a `command` field that specifies the message type.

### 1. Initialization Command

The first message sent after establishing the WebSocket connection must be an initialization command.

#### Request Format

```json
{
  "command": "init",
  "session_id": "session_12345",
  "avatar_id": "16cb73e7de08",
  "quality": "high",
  "version": "v1",
  "video_encoding": "H264",
  "activity_idle_timeout": 120,
  "area": "NORTH_AMERICA",
  "agora_settings": {
    "app_id": "dllkSlkdmmppollalepls",
    "token": "lkmmopplek",
    "channel": "room1",
    "uid": "333",
    "enable_string_uid": false
  }
}
```

#### Root Level Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| command | string | Yes | Must be set to `"init"` for initialization messages |
| session_id | string | Yes | Session identifier received from the initial POST /session/start response. Used to link the WebSocket connection to the session. |
| avatar_id | string | Yes | Unique identifier for the avatar to be used in the session. This ID determines which virtual avatar will be rendered and animated during the video stream. |
| quality | string | Yes | Video quality setting for the avatar stream. Accepted values: `"low"`, `"medium"`, `"high"`. Higher quality settings provide better visual fidelity but require more bandwidth. |
| version | string | Yes | API version identifier. Currently supports `"v1"`. This ensures compatibility between client and server implementations. |
| video_encoding | string | Yes | Video codec to be used for encoding the avatar stream. Supported values: `"H264"`, `"VP8"`, `"AV1"`. H264 provides the widest compatibility across devices and browsers. |
| activity_idle_timeout | number | No | Session timeout in seconds after which the session will be automatically terminated if no activity is detected. Default is 120 seconds. Set to 0 to disable timeout. |
| area | string | No | Geographic hint for avatar provider server selection. The provider can use this to route to nearby infrastructure and minimize latency. Valid values: `"GLOBAL"`, `"NORTH_AMERICA"`, `"EUROPE"`, `"ASIA"`, `"INDIA"`, `"JAPAN"`. Default is `"GLOBAL"`. |
| video_width / video_height | number | No | Optional output size request in pixels; send the **same** values as in `POST /session/start`. See [Video Dimensions](../connection-setup/README.md#video-dimensions-optional). |
| aspect_ratio | string | No | Optional shape request (`"W:H"`) used when exact width/height are not given; send the **same** value as in `/session/start`. |
| agora_settings | object | Yes | Configuration object for Agora RTC (Real-Time Communication) integration. Contains all necessary parameters for establishing the video/audio channel. |

#### Agora Settings Object

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| app_id | string | Yes | Agora application identifier. |
| token | string | Yes | Agora authentication token for secure channel access. |
| channel | string | Yes | Name of the Agora channel to join. |
| uid | string | Yes | User ID within the Agora channel. |
| enable_string_uid | boolean | Yes | Determines whether the uid field should be treated as a string or numeric value. The Golang SDK for publishing back into Agora should be configured with serviceCfg.UseStringUid = enable_string_uid |

#### Vendor Params Passthrough

The init message accepts additional vendor-specific top-level fields beyond
the standard fields listed above. Unknown fields are forwarded to the avatar
provider unchanged. The same vendor-specific fields sent in the session start
request should also be included here — the provider may be stateless between
HTTP session setup and WebSocket connect.

**Reserved (fixed schema):**
`command`, `session_id`, `avatar_id`, `quality`, `version`, `video_encoding`,
`activity_idle_timeout`, `area`, `video_width`, `video_height`, `aspect_ratio`,
`agora_settings`

**Passthrough (vendor-specific):**
Any other top-level key. Vendor-specific params should be top-level, not
nested inside `agora_settings`.

Example with vendor-specific fields:

```json
{
  "command": "init",
  "session_id": "session_12345",
  "avatar_id": "16cb73e7de08",
  "quality": "high",
  "version": "v1",
  "video_encoding": "H264",
  "area": "NORTH_AMERICA",
  "model": "avatar-v3",
  "style": "professional",
  "agora_settings": { ... }
}
```

### 2. Voice Command

After successful initialization, audio data can be streamed using voice commands.

#### Request Format

```json
{
  "command": "voice",
  "audio": "UklGRnoGAABXQVZFZm10IBAAAAABAAEAQB8AAEAfAAABAAgAZGF0YQoGAACBhYqFbF1fdJivrJBhNjVgodDbq2EcBj+a2/LDciUFLIHO8tiJNwgZaLvt559NEAxQp+PwtmMcBjiR1/LMeSwFJHfH8N2QQAoUXrTp66hVFApGn+LyvmASBjqT2fPNeSsFJHfH8N2QQAoUXrTp66hVFApGn+LyvmASBjqT2fPNeSsFJHfH8N2QQAoUXrTp66hVFApGn+LyvmASBjqT2fPNeSsFJHfH8N2QQAoUXrTp66hVFApGn+LyvmASBjqT2fPNeSsFJHfH8N2QQAoUXrTp66hVFApGn+LyvmASBjqT2fPNeSsFJHfH8N2QQAoUXrTp",
  "sampleRate": 24000,
  "encoding": "PCM16",
  "event_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

#### Request Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| command | string | Yes | Must be set to `"voice"` for audio messages |
| audio | string | Yes | Base64-encoded audio data. The audio should be in the format specified by the `encoding` field. |
| sampleRate | number | Yes | Sample rate of the audio data in Hz. Common values: `16000`, `24000`, `44100`, `48000` |
| encoding | string | Yes | Audio encoding format. Supported values: `"PCM16"` (16-bit PCM), `"PCM8"` (8-bit PCM), `"OPUS"` |
| event_id | string | Yes | Unique identifier for this audio chunk. Should be a UUID or similar unique string for tracking purposes. |

### 3. Voice End Command

Signals the end of current speech segment. 

#### Request Format

```json
{
  "command": "voice_end",
  "event_id": "550e8400-e29b-41d4-a716-446655440001"
}
```

#### Request Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| command | string | Yes | Must be set to `"voice_end"` for end-of-speech messages |
| event_id | string | Yes | Unique identifier for this voice_end event. Should be a UUID or similar unique string for tracking purposes. |

### 4. Voice Interrupt Command

Immediately interrupts any ongoing avatar speech. 

#### Request Format

```json
{
  "command": "voice_interrupt",
  "event_id": "550e8400-e29b-41d4-a716-446655440002"
}
```

#### Request Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| command | string | Yes | Must be set to `"voice_interrupt"` for interrupt messages |
| event_id | string | Yes | Unique identifier for this interrupt event. Should be a UUID or similar unique string for tracking purposes. |

### 5. Heartbeat Command

Periodic heartbeat message sent every 10 seconds to maintain connection during periods of no activity.

#### Request Format

```json
{
  "command": "heartbeat",
  "event_id": "550e8400-e29b-41d4-a716-446655440003",
  "timestamp": 1673456789000
}
```

#### Request Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| command | string | Yes | Must be set to `"heartbeat"` for heartbeat messages |
| event_id | string | Yes | Unique identifier for this heartbeat event. Should be a UUID or similar unique string for tracking purposes. |
| timestamp | number | Yes | Unix timestamp in milliseconds when the heartbeat was sent. |

#### Response Format

The server may optionally respond with a heartbeat acknowledgment:

```json
{
  "command": "heartbeat_ack",
  "event_id": "550e8400-e29b-41d4-a716-446655440003",
  "timestamp": 1673456789000
}
```

### 6. Special Command

Sends special instructions or notifications, including avatar actions. The `content`
format depends on the use case.

#### Request Format

```json
{
  "command": "special",
  "content": "XML markup for avatar gestures or other special instructions",
  "event_id": "550e8400-e29b-41d4-a716-446655440005"
}
```

#### Request Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| command | string | Yes | Must be set to `"special"` for special instruction messages |
| content | string | Yes | Special instruction content. Format depends on the specific use case (XML, JSON, etc.) |
| event_id | string | Yes | Unique identifier for this special instruction event. Should be a UUID or similar unique string for tracking purposes. |

#### Avatar Actions

Available in development; not yet released.

ConvoAI sends actions as soon as they are available, independently of audio playback.
Several actions in a paragraph may arrive close together, before the corresponding words
are spoken. The avatar provider decides how overlapping animations are handled.

```json
{
  "command": "special",
  "content": "{\"type\":\"action\",\"action\":\"wave\"}",
  "event_id": "cd3c81e682af4bd5a63d60be07b0b537"
}
```

Decode the outer message, then decode the JSON string in `content`. For `type: "action"`,
handle the gesture named by `action`. The vendor defines the gesture names; ignore unsupported
names and document how actions overlap and stop.

Actions can be triggered in three ways:

- **Inline in an LLM reply:** a text LLM writes `Hello. [wave]`. Configure
  `avatar.params.action_wrapper: "[]"`, `tts.skip_patterns: [4]`, and `tts.batch: true`
  in the join properties, and list the vendor's names in the prompt. The bracketed action
  is removed before text-to-speech and kept in the transcript. `tts.skip_patterns` may be
  omitted -- declaring `action_wrapper` is enough, and the matching pattern is derived --
  but if you do supply the list it must contain the pair, or the request is rejected.
- **Inline text from a developer:** send the same syntax through the `speak` API or typed text
  over RTM. Use the inline configuration above with `parameters.speak.batch: true`.
  No LLM is required.
- **The built-in gesture tool:** a text LLM or speech-to-speech model calls `perform_gesture`
  with `{"action":"wave"}`. Configure `avatar.actions` with the vendor's names and
  `advanced_features.enable_tools: true`. ConvoAI handles the call internally and rejects
  names outside that list. No external tool server or inline wrapper is required.

All three routes send the same frame. Avatar providers implement action handling;
the reference receiver currently demonstrates audio and control messages only.

## Testing

### Prerequisites
Before running the tests, you need an audio file:
1. Create or obtain a WAV file named `input.wav` in the `websocket-receive-audio` directory
2. The WAV file should be PCM 16-bit format for best compatibility

### Steps to Run the Test

To test the WebSocket API locally:

1. **Start the test receiver**:
   ```bash
   python websocket_test_receiver.py
   ```
   This starts a WebSocket server on `ws://localhost:8765` that will:
   - Validate session tokens
   - Log all received commands
   - Save received audio as `received_audio.wav`

2. **In a new terminal, run the audio sender**:
   ```bash
   python websocket_audio_sender.py
   ```
   This will:
   - Connect to the WebSocket server
   - Send an `init` command with session configuration including session_id
   - Stream audio chunks from `input.wav`
   - Send a `voice_end` command when complete

3. **Verify the test**: 
   - Check the receiver terminal for logged messages
   - Verify that `received_audio.wav` is created in your directory
   - Compare the original `input.wav` with `received_audio.wav`

### Testing Against Your Own Implementation

To test against your own WebSocket server instead of the mock receiver:

1. **Update the WebSocket address** in `websocket_audio_sender.py`:
   ```python
   WEBSOCKET_ADDRESS = "wss://your-api.com/v1/websocket"
   ```

2. **Update the session token** with a real token from your connection setup API:
   ```python
   SESSION_TOKEN = "your_real_session_token_here"
   ```

3. **Update configuration values** as needed:
   ```python
   APP_ID = "your_agora_app_id"
   TOKEN = "your_agora_token"  
   CHANNEL = "your_channel_name"
   UID = "your_user_id"
   AVATAR_ID = "your_avatar_id"
   ```

4. Run the sender script normally

### Testing Notes
- The mock receiver expects session token `test_session_token_12345` by default
- Tests cover essential commands: `init` (including session_id), `voice`, `voice_end`, and `heartbeat`
- All commands are logged with detailed information for debugging
- Audio is saved in PCM 16-bit WAV format
- The sender demonstrates the core message flow needed for audio streaming with proper session identification
