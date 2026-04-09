# 01 Setup

> Local setup, required tools, and fastest path to run each component.

## Tooling

- Python 3.11+ for the Python examples and tests
- Go 1.21 for `go-publish-video/`
- `flatbuffers-compiler` for `go-publish-video/ipc`
- Agora Go Server SDK checkout for the Go publisher

## Runtime Matrix

| Component | Language | Runs Standalone | Needs Agora Credentials |
| --- | --- | --- | --- |
| `connection-setup/` | Python | Yes | No |
| `websocket-receive-audio/` | Python | Yes | No |
| `go-publish-video/` tests | Go | Yes | No |
| `go-publish-video/` live publish | Go + native SDK | Yes | Yes |

## Repo Entry Points

- repo overview: `README.md`
- root agent instructions: `AGENTS.md`
- Go publisher instructions: `go-publish-video/README.md`
- Python mock docs:
  - `connection-setup/README.md`
  - `websocket-receive-audio/README.md`

## Python Components

- `connection-setup/`
  - install: `pip install -r requirements.txt -r requirements-dev.txt`
  - test: `pytest -v`
- `websocket-receive-audio/`
  - install: `pip install -r requirements.txt -r requirements-dev.txt`
  - test: `pytest -v`

## Go Publisher

- work in `go-publish-video/`
- branch workaround target:
  - Go SDK `v2.2.4`
  - native Linux RTC `4.4.31`
- set local SDK path:
  - `go mod edit -replace github.com/AgoraIO-Extensions/Agora-Golang-Server-SDK/v2=/path/to/Agora-Golang-Server-SDK`
- copy native libs:
  - Linux/macOS: `cp -r /path/to/Agora-Golang-Server-SDK/agora_sdk ./`
- Linux run env:
  - `export LD_LIBRARY_PATH=$(pwd)/agora_sdk:$LD_LIBRARY_PATH`

## Recommended SDK Line

- workaround branch: `linux_rtc_4_4_31`
- Go wrapper target: `v2.2.4`
- native Linux RTC target: `4.4.31`
- rationale:
  - this path was verified on Modal/gVisor
  - newer `4.4.32` was the failing line during startup on Modal

## Build Outputs

- `make build` creates:
  - `go-publish-video/parent`
  - `go-publish-video/child`
- `make test` runs:
  - IPC framing tests
  - parent loop/unit tests
  - child-related unit tests that do not require a live Agora session

## Common Commands

```bash
cd go-publish-video
make build
make test
./parent -appID "<app_id>" -channelName "<channel>" -userID "10" -enableStringUID=false
```

## Fastest Manual Test

1. build the publisher in `go-publish-video/`
2. run `./parent` with numeric UID mode
3. open a browser-capable Agora demo in the same channel
4. join with a different numeric viewer UID

## Important Flags

| Flag | Purpose |
| --- | --- |
| `-appID` | Agora App ID |
| `-channelName` | target channel |
| `-token` | optional publish token |
| `-userID` | publisher UID |
| `-enableStringUID` | choose string vs numeric UID mode |
| `-videoFile` | raw sample video file |
| `-audioFile` | PCM sample audio file |
| `-width` / `-height` | raw frame dimensions |
| `-frameRate` | pacing for the video loop |

## Sample Media

- audio sample: `go-publish-video/test_data/send_audio_16k_1ch.pcm`
- original YUV sample: `go-publish-video/test_data/send_video_cif.yuv`
- Modal workaround sample: `go-publish-video/test_data/send_video_cif.rgb24`

## File Format Expectations

- `send_audio_16k_1ch.pcm`
  - PCM16
  - mono
  - 16 kHz
- `send_video_cif.rgb24`
  - packed `RGB24`
  - `352x288`
  - 3 bytes per pixel
- parent converts `RGB24` to `RGBA` before sending to child

## Common Setup Failures

- missing FlatBuffers binary: `make build` fails during IPC generation
- missing Agora SDK headers/libs: Go CGO build fails
- missing `LD_LIBRARY_PATH` on Linux: binary starts but native libs fail to load
- wrong SDK line on Modal: newer Linux RTC `4.4.32` path is not the workaround target

## Validation Checklist

- `go mod edit -replace ...` points to a real local SDK checkout
- `agora_sdk/` exists inside `go-publish-video/`
- `LD_LIBRARY_PATH` includes `$(pwd)/agora_sdk` on Linux
- `parent` and `child` both build
- viewer joins with matching UID mode
- tokens match the same UID mode used by the publisher

## Related Deep Dives

- [Go Publisher Pipeline](deep_dives/go_publisher_pipeline.md) — parent/child media flow and frame handling
- [Modal SDK Compatibility](deep_dives/modal_sdk_compatibility.md) — known-good SDK line and failure boundary
