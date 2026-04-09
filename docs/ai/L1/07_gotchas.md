# 07 Gotchas

> Critical pitfalls, environment-specific behavior, and historical failure modes.

## Modal / gVisor

- newer Linux RTC `4.4.32` was the failing line investigated for Modal
- workaround branch pins to `4.4.31`
- do not silently upgrade the native Linux SDK on this branch

## SDK Version Boundary

- known-good Modal workaround line:
  - Go wrapper `v2.2.4`
  - native Linux RTC `4.4.31`
- known-bad Modal line investigated:
  - native Linux RTC `4.4.32`
- if a Modal regression reappears, check the installed native SDK first

## UID Mode

- browser testing usually requires `-enableStringUID=false`
- all participants must use the same UID mode
- earlier parent flag passing could mis-handle this; current workaround branch fixes that

## Viewer Confusion

- a voice-only client can make a healthy publish look broken because you hear audio but see no video
- re-check with a video-capable viewer before debugging the publisher path

## Media Format Mismatch

- `RGB24` input and `RGBA` publish are not interchangeable
- width/height and bytes-per-frame must match the file exactly
- partial frame reads cause loop/reset behavior

## Timestamp Pitfall

- earlier child code could drop parent timestamps
- current branch propagates timestamps through IPC to publish calls
- if sync looks wrong, check parent sample timestamps before child send code

## Python Components

- Python services are mocks and protocol examples
- they do not automatically feed the Go publisher

## Build Expectations

- the Agora Go SDK needs its native `agora_sdk/` directory present
- a plain Go module download is not enough for CGO builds

## Runtime Artifacts

- `agora_child_sdk.log` and `agoraapi.log` can help when the child starts
- a failure before SDK init may produce no useful SDK file log at all

## Sample Asset Pitfalls

- the bundled RGB sample is sized for the current default width and height
- swapping in another raw file without matching flags leads to broken frames
- YUV and RGB sample files are not interchangeable even if resolution matches

## Manual Test Pitfalls

- reusing the publisher UID in the viewer creates confusing behavior
- stale browser tabs can remain joined to an older channel
- testing with the wrong token type can look like a media bug when it is really an auth mismatch

## First Things To Check

1. UID mode matches across publisher and viewer
2. SDK line is still `4.4.31` on the workaround branch
3. bytes-per-frame math matches the chosen raw input format
4. browser test is done with a video-capable viewer
5. sample file dimensions match the CLI flags
6. `agora_sdk/` was copied from the intended local SDK checkout

## Escalate When

- direct native samples fail on the same environment
- SDK creation or init fails before a channel join
- the workaround branch breaks without a code change in this repo

## Related Deep Dives

- [Modal SDK Compatibility](deep_dives/modal_sdk_compatibility.md) — what was tested and where the failure boundary was
- [Go Publisher Pipeline](deep_dives/go_publisher_pipeline.md) — where timestamp and pixel-format bugs showed up
