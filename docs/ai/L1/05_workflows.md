# 05 Workflows

> Step-by-step common tasks for working in this repo.

## Run The Go Publisher Locally

1. install Go, FlatBuffers, and local Agora SDK checkout
2. set the `go.mod` replace directive to the local Agora SDK path
3. copy `agora_sdk/` into `go-publish-video/`
4. run `make build`
5. run `./parent ...`

## Validate With Browser Viewer

1. run publisher with `-enableStringUID=false`
2. open a browser-capable Agora demo
3. join the same channel with a different numeric UID
4. confirm both audio and video render
5. if audio works but video does not, inspect format and viewer UID mode first

## Swap Sample Media

1. put the audio/video files in `go-publish-video/test_data/`
2. pass `-audioFile` and `-videoFile`
3. ensure width, height, frame rate, and format match the file content

## Change The Video Input Format

1. decide the file format at the parent boundary
2. update bytes-per-frame math in `parent.go`
3. update conversion logic before IPC send
4. update child pixel-format constant if SDK input changes
5. update `README.md` and `docs/ai/`
6. rerun a live browser verification

## Test Numeric UID Browser Flow

1. run publisher with `-enableStringUID=false`
2. join the same channel from a browser-capable Agora demo
3. use a different viewer UID

## Test String UID Flow

1. run publisher with `-enableStringUID=true`
2. use a subscriber or app that supports string UIDs
3. generate the matching token type
4. do not use the basic numeric-only web demo for this path

## Update The Modal Workaround Branch

1. stay on branch `linux_rtc_4_4_31`
2. keep SDK docs pinned to the `4.4.31` line
3. verify with a real Modal run before calling the change complete

## Check The Current Publisher Path

1. inspect `go-publish-video/parent.go`
2. inspect `go-publish-video/child.go`
3. confirm input format and SDK output format still match

## Debug Startup Failures

1. confirm local SDK checkout and `go mod edit -replace`
2. confirm `agora_sdk/` is present in `go-publish-video/`
3. confirm `LD_LIBRARY_PATH` on Linux
4. run a bounded local test before blaming the media loop
5. if failure is Modal-specific, compare against the pinned `4.4.31` path

## Update Progressive Disclosure Docs

1. update the affected L1 summary files
2. add or revise a deep dive only when L1 becomes too compressed
3. keep `AGENTS.md` aligned with the docs entrypoint
4. mention environment-specific gotchas in `07_gotchas.md`

## Useful Verification Signals

- parent reports that the child connected
- child logs show local audio and video tracks published
- repeated send logs return success
- browser viewer renders both tracks with matching UID mode

## Related Deep Dives

- [Go Publisher Pipeline](deep_dives/go_publisher_pipeline.md) — exact publish loop behavior
- [Modal SDK Compatibility](deep_dives/modal_sdk_compatibility.md) — known-good version matrix and Modal notes
