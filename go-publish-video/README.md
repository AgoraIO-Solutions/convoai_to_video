# Publish Audio and Video into Agora with Golang (v2.4.12)

This document guides you through setting up and publishing YUV video frames and PCM audio into an Agora channel using the Agora Golang SDK v2.4.12.
The steps have been verified on Ubuntu 24.04 and macOS but should be compatible with other Debian/Ubuntu versions.
## Why Parent/Child Processes?

`parent.go` launches `child.go` as a separate process and communicates with it using FlatBuffers over IPC. This design is intentional:

- **Avoids SDK re-initialization issues** — the Agora C SDK does not cleanly re-initialize within the same process. By isolating it in a child process, each call simply spawns a new child rather than attempting to tear down and re-init the SDK in-process.
- **Thread safety** — CGo and the Agora C SDK's internal threading model are isolated from the main application, avoiding conflicts.
- **Stability** — if the native SDK crashes, it takes down the child process, not your entire application.

## Key Features (v2.4.12)
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

> **`go.mod` carries a local `replace`** pointing the Agora SDK at an absolute
> path on the maintainer's machine. Repoint it at your own checkout, or delete
> the line to use the published module, before building outside that machine.

> **AV1 minimum versions:** Go SDK >= v2.4.12, native SDK >= v4.4.32.165.
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

## Colour space — set it, or the picture goes green

`ExternalVideoFrame.ColorSpace` is a **struct**, so omitting it does not mean
"unspecified". Go zero-values it and the SDK copies all four fields to the native
frame, and in H.273 the matrix index `0` means **GBR** — you have told the receiver
your three I420 planes are Green, Blue and Red.

This publisher therefore always declares it, the same way for every codec:

```go
extFrame := &agoraservice.ExternalVideoFrame{
    Type:      agoraservice.VideoBufferRawData,
    Format:    agoraservice.VideoPixelI420,
    Buffer:    frameData,
    Stride:    int(width),
    Height:    int(height),
    Timestamp: timestampMs,
    ColorSpace: agoraservice.ColorSpaceType{
        RangeId:     0, // unspecified
        MatrixId:    2, // unspecified -- NOT 0, which means GBR
        PrimariesId: 2, // unspecified
        TransferId:  2, // unspecified
    },
}
rtcConnection.PushVideoFrame(extFrame)
```

`2` means "unspecified", so the receiver applies its own default — which is what it
was already doing while the frames were untagged. Naming a real matrix instead is a
guess about pixel data this process never inspected, and when the guess disagrees
with the receiver's default it darkens the picture (measured at −3.3 luma and −4.2
green on 640×360 content). Only declare one if you actually know your source.

What goes wrong without it differs by codec, and both were measured in a browser:

| Codec | With matrix 0 (unset) |
| --- | --- |
| H264 / VP8 | A legal but false statement, so the picture goes green — and only once something honours it. Hardware decoders usually ignore the signalled matrix, so it appears when a renegotiation drops the decoder to its software path. That is why it shows up on mute rather than at the start of a call. |
| AV1 | Not merely wrong but **non-conformant**: the AV1 spec requires 4:4:4 when `matrix_coefficients` is `MC_IDENTITY` (0), and this publisher sends I420 (4:2:0). Chrome tolerates the invalid stream and renders GBR immediately; a stricter decoder may refuse it. |

The publisher logs what it declared, so you can check without a wire probe:

```
Declared colour space: matrix=2 range=0 primaries=2 transfer=2 (codec=AV1; ...)
```

Override with `-colorMatrix` / `-colorRange` / `-colorPrimaries` / `-colorTransfer`,
or the `AVATAR_COLOR_*` environment variables.

### On the AV1 note in the Go SDK README

The Go SDK's own README asks for `(PrimariesId 2, TransferId 2, MatrixId 5, RangeId 1)`
for AV1 and says web decoding "will not work" without it. We could not reproduce that:
AV1 with everything unspecified decodes and renders correctly in Chrome (Web SDK
4.24.9), including across a mute/unmute renegotiation. The failure that note describes
is almost certainly the zero value — matrix 0, the non-conformant case above — for
which `(2,2,5,1)` is *a* conformant fix rather than the only one.

If a receiver is ever reported failing to decode AV1, that tuple is the first thing to
try. It is an alternative, not a contradiction.

The same SDK note adds that with RTC SDK `tag=163`, a non-nil `MetadataBuffer` carrying
SEI also stops the web client decoding AV1. This publisher never sets one. Unlike
`ColorSpace`, `MetadataBuffer` is a slice, so leaving it out genuinely is nil.

## Testing

No unit-test suite is included. `make test` reports that gap and exits non-zero.

### Agora Audio Smoke Check

The smoke check requires the native SDK and valid credentials. It runs both
string and numeric UID modes with matching tokens and checks for at least
10 non-empty audio frames from the expected publisher. The subscriber does
not receive video or check A/V synchronization; verify those with a viewer.

```bash
# Build the smoke-check binaries
make build
make subscriber
make tokengen

# Run the audio smoke check
APP_ID=<your_app_id> APP_CERT=<your_app_cert> make test-e2e
```

The `tokengen` tool is a standalone binary in `cmd/tokengen/` with its own `go.mod` — it has no Agora SDK dependency and can be built on any platform.

### Manual Testing

To manually publish and view the stream:

```bash
# Generate a publisher token (if app cert is enabled)
PUB_TOKEN=$(./tokengen -appID <app_id> -appCert <app_cert> \
  -channelName <channel> -uid <uid> -role publisher -enableStringUID=false)

# Start publisher. -enableStringUID must match the token: parent defaults to
# true, and the token above was generated with false.
./parent -appID <app_id> -token "$PUB_TOKEN" \
  -channelName <channel> -userID <uid> -videoCodec AV1 -enableStringUID=false

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
  - Requires Go SDK >= v2.4.12 and native SDK >= v4.4.32.165
  - Recommended bitrate: 1500-2000 Kbps
  - Recommended min bitrate: 500-800 Kbps
  - If the receiver shows H264 despite requesting AV1, ensure the native SDK is up to date (`bash scripts/install_agora_sdk.sh`)

## Makefile Targets

| Target | Description |
|--------|-------------|
| `make` | Generate FlatBuffers code and build binaries |
| `make build` | Build child and parent binaries |
| `make test` | Report the missing unit-test suite and exit non-zero |
| `make clean` | Remove build artifacts |
| `make subscriber` | Build subscriber binary (for e2e tests) |
| `make tokengen` | Build token generator (for e2e tests) |
| `make test-e2e` | Run the credentialed Agora audio smoke check |
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
