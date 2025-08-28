# WebSocket API Documentation

This document describes the WebSocket API for driving remote audio and video generation sessions with streaming audio.

## Connection Setup

### Headers

Required headers for establishing the WebSocket connection:

```json
{
  "accept": "application/json",
  "content-type": "application/json",
  "authorization": "Bearer {session_token}"
}
```

#### Header Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| authorization | string | Yes | Bearer token authentication. Format: `Bearer {session_token}` where `{session_token}` is the token obtained from the initial connection setup endpoint. 

## Message Protocol

All messages are sent as JSON strings over the WebSocket connection. The API uses a command-based protocol where each message contains a `command` field that specifies the message type.

### 1. Initialization Command

The first message sent after establishing the WebSocket connection must be an initialization command.

#### Request Format

```json
{
  "command": "init",
  "avatar_id": "16cb73e7de08",
  "quality": "high",
  "version": "v1",
  "video_encoding": "H264",
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
| avatar_id | string | Yes | Unique identifier for the avatar to be used in the session. This ID determines which virtual avatar will be rendered and animated during the video stream. |
| quality | string | Yes | Video quality setting for the avatar stream. Accepted values: `"low"`, `"medium"`, `"high"`. Higher quality settings provide better visual fidelity but require more bandwidth. |
| version | string | Yes | API version identifier. Currently supports `"v1"`. This ensures compatibility between client and server implementations. |
| video_encoding | string | Yes | Video codec to be used for encoding the avatar stream. Supported values: `"H264"`, `"VP8"`, `"AV1"`. H264 provides the widest compatibility across devices and browsers. |
| agora_settings | object | Yes | Configuration object for Agora RTC (Real-Time Communication) integration. Contains all necessary parameters for establishing the video/audio channel. |

#### Agora Settings Object

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| app_id | string | Yes | Agora application identifier. |
| token | string | Yes | Agora authentication token for secure channel access. |
| channel | string | Yes | Name of the Agora channel to join. |
| uid | string | Yes | User ID within the Agora channel. |
| enable_string_uid | boolean | Yes | Determines whether the uid field should be treated as a string or numeric value. The Golang SDK for publishing back into Agora should be configured with serviceCfg.UseStringUid = enable_string_uid |

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

## Testing

### Steps to Run the Test
To run the test locally you can use the scripts as they are below.     
To send audio to your own websocket edit the websocket address in websocket_audio_sender.py        
1. **Start the test receiver**:
   ```bash
   python websocket_test_receiver.py
   ```

2. **In a new terminal, run the audio sender**:
   ```bash
   python websocket_audio_sender.py
   ```

3. **Verify the test**: Check that `received_audio.wav` is created in your directory after the sender completes.
