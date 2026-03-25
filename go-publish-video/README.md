# Publish Audio and Video into Agora with Golang (v2.4.11)

This document guides you through setting up and publishing YUV video frames and PCM audio into an Agora channel using the Agora Golang SDK v2.4.11.
The steps have been verified on Ubuntu 24.04 and macOS but should be compatible with other Debian/Ubuntu versions.
parent.go launches a child.go in its own process and communicates with it using IPC. This ensures efficient movement of data while keeping each call in its own process for stability and threading optimisation.

## Key Features (v2.4.11)
- Support for multiple video codecs: H264, VP8, and AV1
- Simplified SDK API with direct push methods for audio/video
- Local SDK library integration without system-wide installation
- Enhanced IPC communication using FlatBuffers

## Installation Steps

### 1. Install Build Essentials and Go

**Linux (Ubuntu/Debian):**
```bash
sudo apt-get update
sudo apt-get install -y build-essential git wget unzip flatbuffers-compiler

# Download and install Go 1.21 (if not already installed)
wget https://go.dev/dl/go1.21.13.linux-amd64.tar.gz
sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf go1.21.13.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin
```

**macOS:**
```bash
# Install Go via Homebrew (if not already installed)
brew install go flatbuffers
```

### 2. Setup Agora Go SDK (v2.4.10+ required)

```bash
# Clone the Agora Golang Server SDK
git clone https://github.com/AgoraIO-Extensions/Agora-Golang-Server-SDK.git
cd Agora-Golang-Server-SDK

# Use latest main (must be v2.4.10+ for AV1 support)
git checkout main && git pull origin main

# Download native SDK libraries (headers + .so/.dylib files)
bash scripts/install_agora_sdk.sh
```

> **AV1 minimum versions:** Go SDK >= v2.4.10, native SDK >= v4.4.32.164 (Feb 2026).
> Older native SDKs silently fall back to H264 when AV1 is requested — no error is reported.

### 3. Clone and Setup This Project

```bash
git clone https://github.com/AgoraIO-Solutions/convoai_to_video.git
cd convoai_to_video/go-publish-video
```

Update the `replace` directive in `go.mod` to point to your local SDK checkout:
```
replace github.com/AgoraIO-Extensions/Agora-Golang-Server-SDK/v2 => /path/to/Agora-Golang-Server-SDK
```

**Linux only** — copy the SDK native libraries into the project:
```bash
cp -r /path/to/Agora-Golang-Server-SDK/agora_sdk ./
```

On macOS the SDK's own cgo directives handle library linking automatically.

### 4. Build and Run

```bash
# Build everything (generates FlatBuffers code, builds parent + child binaries)
make

# Run with your Agora credentials
./parent -appID "<your_app_id>" -channelName "<your_channel>" -userID "<your_uid>"

# With VP8 codec
./parent -appID "<your_app_id>" -channelName "<your_channel>" -videoCodec "VP8"

# With AV1 codec
./parent -appID "<your_app_id>" -channelName "<your_channel>" -videoCodec "AV1"

# With integer user IDs
./parent -appID "<your_app_id>" -channelName "<your_channel>" -userID "12345" -enableStringUID=false
```

**Linux only** — set the library path before running:
```bash
export LD_LIBRARY_PATH=$(pwd)/agora_sdk:$LD_LIBRARY_PATH
```

This will publish the YUV and PCM files from the test_data folder. You can view the stream on Agora Web Demo:
https://webdemo.agora.io/basicVideoCall/index.html

Use your App ID and Channel Name to join the stream.

## Testing

### Unit Tests

Unit tests cover FlatBuffers IPC round-trips, parent/child message framing, and codec configuration logic. They do **not** require Agora SDK native libraries or a network connection.

```bash
make test
```

### E2E Integration Tests

End-to-end tests verify that the publisher can connect, publish audio/video, and that a subscriber receives audio frames. Tests both string UID and numeric UID modes with token authentication.

```bash
# Build the test binaries
make build
make subscriber
make tokengen

# Run e2e tests (requires valid Agora credentials)
APP_ID=<your_app_id> APP_CERT=<your_app_cert> make test-e2e
```

The `tokengen` tool is a standalone binary in `cmd/tokengen/` with its own `go.mod` — it has no Agora SDK dependency and can be built on any platform.

### Manual Testing

To manually publish and view the stream:

```bash
# Generate a publisher token (if app cert is enabled)
PUB_TOKEN=$(./tokengen -appID <app_id> -appCert <app_cert> \
  -channelName <channel> -uid <uid> -role publisher -enableStringUID=false)

# Start publisher
./parent -appID <app_id> -token "$PUB_TOKEN" \
  -channelName <channel> -userID <uid> -videoCodec AV1

# Generate a subscriber token for the web viewer
./tokengen -appID <app_id> -appCert <app_cert> \
  -channelName <channel> -uid <viewer_uid> -role subscriber -enableStringUID=false
```

Join at https://webdemo.agora.io/basicVideoCall/index.html with the App ID, channel, viewer UID, and subscriber token.

## Key Parameters

**Required:**
- `-appID`: Your Agora Application ID
- `-channelName`: Channel name to join

**Video Codec:**
- `-videoCodec`: Choose "H264", "VP8", or "AV1" (default: "H264")

**Optional:**
- `-userID`: User ID for the session (default: "100")
- `-token`: Authentication token if required
- `-enableStringUID`: Use string user IDs instead of integers (default: true)
- `-width`, `-height`: Video resolution (default: 352x288)
- `-frameRate`: Video frame rate (default: 15 fps)
- `-bitrate`: Video bitrate in Kbps (default: 1000)

## Codec Notes

- **H264**: Most widely supported, good balance of quality and performance
- **VP8**: Open source codec, good for web compatibility
- **AV1**: Latest generation codec, better compression but requires more CPU
  - Requires Go SDK >= v2.4.10 and native SDK >= v4.4.32.164
  - Recommended bitrate: 1500-2000 Kbps
  - Recommended min bitrate: 500-800 Kbps
  - If the receiver shows H264 despite requesting AV1, update the native SDK (`bash scripts/install_agora_sdk.sh`)

## Makefile Targets

| Target | Description |
|--------|-------------|
| `make` | Generate FlatBuffers code and build binaries |
| `make build` | Build child and parent binaries |
| `make test` | Run Go unit tests |
| `make clean` | Remove build artifacts |
| `make subscriber` | Build subscriber binary (for e2e tests) |
| `make tokengen` | Build token generator (for e2e tests) |
| `make test-e2e` | Run e2e integration tests |
| `make run APP_ID=<id>` | Run the demo |
| `make help` | Show all targets |

## Troubleshooting

If you encounter build errors:
1. Ensure the Agora SDK is cloned and up to date (`git pull origin main`)
2. Re-run `bash scripts/install_agora_sdk.sh` to get the latest native SDK
3. Verify the `replace` directive in `go.mod` points to your SDK checkout
4. On Linux, ensure `agora_sdk/` contains the required `.so` files
5. On macOS, ensure the SDK checkout contains `agora_sdk_mac/` with `.dylib` files

## Next Steps

Modify parent.go to send your own YUV video and PCM audio into Agora. Publish them together in sync and in realtime.
