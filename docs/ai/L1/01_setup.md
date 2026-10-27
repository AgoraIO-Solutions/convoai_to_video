# 01 Setup

> Local setup, required tools, and fastest path to run each component.

## Tooling

- Python 3.11+ for the Python examples and manual checks
- Go 1.21 for `go-publish-video/`
- `flatbuffers-compiler` for `go-publish-video/ipc`
- Agora Go Server SDK checkout for the Go publisher

## Runtime Matrix

| Component | Language | Runs Standalone | Needs Agora Credentials |
| --- | --- | --- | --- |
| `connection-setup/` | Python | Yes | No |
| `websocket-receive-audio/` | Python | Yes | No |
| `go-publish-video/` audio smoke check | Go + native SDK | Yes | Yes |
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
  - install: `pip install -r requirements.txt`
  - start `session_test_receiver.py`, then run `session_start.py` and `session_stop.py`
  - clients exit non-zero if any check fails
- `websocket-receive-audio/`
  - install: `pip install -r requirements.txt`
  - no unit tests: run the pair by hand, receiver first, then sender
    (see that directory's README)

## Go Publisher

- work in `go-publish-video/`
- Go SDK `v2.4.12`, per `go-publish-video/go.mod`
- native Linux RTC version floors live in `go-publish-video/README.md`;
  they are not restated here
- set local SDK path:
  - `go mod edit -replace github.com/AgoraIO-Extensions/Agora-Golang-Server-SDK/v2=/path/to/Agora-Golang-Server-SDK`
- copy native libs:
  - Linux/macOS: `cp -r /path/to/Agora-Golang-Server-SDK/agora_sdk ./`
- Linux run env:
  - `export LD_LIBRARY_PATH=$(pwd)/agora_sdk:$LD_LIBRARY_PATH`

## Recommended SDK Line

- workaround branch: `linux_rtc_4_4_31`
- Go wrapper: `v2.4.12` (`go-publish-video/go.mod`)
- native Linux RTC floors: see `go-publish-video/README.md`
- rationale:
  - this path was verified on Modal/gVisor
  - newer `4.4.32` was the failing line during startup on Modal

## Build Outputs

- `make build` creates:
  - `go-publish-video/parent`
  - `go-publish-video/child`
- no unit-test suite is included; `make test` reports this and exits non-zero
- `make test-e2e` builds and runs an audio smoke check with SDK and credentials
- the smoke check requires 10 non-empty audio frames from the expected publisher
  in each UID mode; video and A/V synchronization need manual verification

## Common Commands

```bash
cd go-publish-video
make build
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
- video sample: `go-publish-video/test_data/send_video_cif.yuv`

Those are the only two assets in `test_data/`. A `send_video_cif.rgb24` was
listed here but has never existed on master.

## File Format Expectations

- `send_audio_16k_1ch.pcm`
  - PCM16
  - mono
  - 16 kHz
- `send_video_cif.yuv`
  - `I420` (YUV420 planar)
  - `352x288`
  - 1.5 bytes per pixel
- parent sends raw `I420` bytes to child; no colour conversion

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
