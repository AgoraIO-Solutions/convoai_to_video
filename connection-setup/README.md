# Connection Setup API Documentation
This document describes the REST API endpoints you should create for managing video generation sessions and providing WebSocket connection details to Agora convoAI platform.

## Endpoints Overview
- `POST /session/start` - Start a new session
- `DELETE /session/stop` - Stop an existing session

## Start Session Endpoint
```
POST /session/start
```

## Headers
```json
{
  "accept": "application/json",
  "content-type": "application/json",
  "x-api-key": "YOUR_API_KEY"
}
```

## Request Format
```json
{
  "avatar_id": "16cb73e7de08",
  "quality": "high",
  "version": "v1",
  "video_encoding": "H264",
  "activity_idle_timeout": 120,
  "agora_settings": {
    "app_id": "dllkSlkdmmppollalepls",
    "token": "lkmmopplek",
    "channel": "room1",
    "uid": "333",
    "enable_string_uid": false
  }
}
```

## Request Fields

### Headers
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| x-api-key | string | Yes | API authentication key for accessing the service. Passed in the request header for security. |

### Body Fields
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| avatar_id | string | Yes | Unique identifier for the avatar to be used in the session. This ID determines which virtual avatar will be rendered and animated during the video stream. |
| quality | string | Yes | Video quality setting for the avatar stream. Accepted values: `"low"`, `"medium"`, `"high"`. Higher quality settings provide better visual fidelity but require more bandwidth. |
| version | string | Yes | API version identifier. Currently supports `"v1"`. This ensures compatibility between client and server implementations. |
| video_encoding | string | Yes | Video codec to be used for encoding the avatar stream. Supported values: `"H264"`, `"VP8"`, `"AV1"`. H264 provides the widest compatibility across devices and browsers. |
| activity_idle_timeout | number | No | Session timeout in seconds after which the session will be automatically terminated if no activity is detected. Default is 120 seconds. Set to 0 to disable timeout. |
| agora_settings | object | Yes | Configuration object for Agora RTC (Real-Time Communication) integration. Contains all necessary parameters for establishing the video/audio channel. |

### Agora Settings Object
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| app_id | string | Yes | Agora application identifier. |
| token | string | Yes | Agora authentication token for secure channel access. |
| channel | string | Yes | Name of the Agora channel to join. |
| uid | string | Yes | User ID within the Agora channel. |
| enable_string_uid | boolean | Yes | Determines whether the uid field should be treated as a string or numeric value. |

## Response Format

### Success Response (200 OK)
```json
{
  "session_id": "550e8400-e29b-41d4-a716-446655440000",
  "websocket_address": "wss://api.example.com/v1/websocket",
  "session_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"
}
```

### Response Fields
| Field | Type | Description |
|-------|------|-------------|
| session_id | string | Unique identifier for this session. Used for session management and stopping the session later. |
| websocket_address | string | WebSocket URL to connect to for audio streaming. Use this address to establish the WebSocket connection. |
| session_token | string | JWT token for WebSocket authentication. Include this in the WebSocket connection headers as `Authorization: Bearer {session_token}`. |

### Error Response (400 Bad Request)
```json
{
  "error": "Invalid request",
  "message": "Missing required field: avatar_id",
  "code": "VALIDATION_ERROR"
}
```

### Error Response (401 Unauthorized)
```json
{
  "error": "Unauthorized", 
  "message": "Invalid API key",
  "code": "INVALID_API_KEY"
}
```

### Error Response (403 Forbidden)
```json
{
  "error": "Forbidden",
  "message": "API key header missing",
  "code": "MISSING_API_KEY"
}
```

## Usage Flow
1. **Send POST request** to `/session/start` with session configuration and API key in header
2. **Receive response** containing `session_id`, `websocket_address` and `session_token`
3. **Connect to WebSocket** using the provided address and token
4. **Send init command** via WebSocket with the same configuration
5. **Stream audio data** using voice commands
6. **Send DELETE request** to `/session/stop` when finished to clean up resources

---

## Stop Session Endpoint
```
DELETE /session/stop
```

### Headers
```json
{
  "accept": "application/json",
  "content-type": "application/json",
  "x-api-key": "YOUR_API_KEY"
}
```

### Request Format
```json
{
  "session_id": "550e8400-e29b-41d4-a716-446655440000",
  "session_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"
}
```

### Request Fields
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| session_id | string | Yes | The session ID received from the `/session/start` endpoint. Used to identify which session to terminate. |
| session_token | string | Yes | The session token received from the `/session/start` endpoint. Used for authentication and session validation. |

### Response Format

#### Success Response (200 OK)
```json
{
  "status": "success",
  "message": "Session terminated successfully"
}
```

#### Error Response (400 Bad Request)
```json
{
  "error": "Invalid request",
  "message": "Missing required field: session_id",
  "code": "VALIDATION_ERROR"
}
```

#### Error Response (401 Unauthorized)
```json
{
  "error": "Unauthorized", 
  "message": "Invalid API key",
  "code": "INVALID_API_KEY"
}
```

#### Error Response (404 Not Found)
```json
{
  "error": "Not found",
  "message": "Session not found or already terminated",
  "code": "SESSION_NOT_FOUND"
}
```

---

## Security Notes
- The API key is now passed in the `x-api-key` header instead of the request body for better security
- This prevents the API key from being logged in request bodies or appearing in URL parameters
- Always use HTTPS in production to protect the API key in transit

## Testing
Use the provided test scripts to verify the endpoints and WebSocket functionality:

### Mock Server
First, start the mock server for local testing:
```bash
python session_test_receiver.py
```
This will start a mock server on `http://localhost:8764` that simulates the session management endpoints.

### Individual Component Testing
```bash
# Test session start endpoint (requires mock server running)
python session_start.py

# Test session stop endpoint (requires mock server running)
python session_stop.py
```

### Complete Session Flow Testing
```bash
# Start mock server in background, then test the full flow
python session_test_receiver.py &
python session_start.py && python session_stop.py

# Or run sequentially in separate terminals:
# Terminal 1: python session_test_receiver.py
# Terminal 2: python session_start.py && python session_stop.py
```

### Testing Against Your Own Implementation
To test against your own API implementation instead of the mock server:

1. Update the `API_ENDPOINT` variable in both test scripts:
   - In `session_start.py`: Change to your start endpoint URL
   - In `session_stop.py`: Change to your stop endpoint URL

2. Update the `API_KEY` variable with your actual API key

3. Run the test scripts normally

### Testing Notes
- The mock server (`session_test_receiver.py`) uses API key `test-api-key-123` by default
- You can set a custom API key via environment variable: `export TEST_API_KEY='your-custom-key'`
- All test scripts include comprehensive validation and error handling tests
- The mock server provides detailed logging for debugging