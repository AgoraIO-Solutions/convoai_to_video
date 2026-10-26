# Go Publisher Pipeline

## Purpose

Deep dive for the `go-publish-video/` process split, media loop, and the current Modal workaround path.

## Process Model

- `parent.go`
  - parses flags
  - opens media files
  - loops sample audio/video
  - reads raw `I420` (YUV420) frames
  - writes FlatBuffers IPC messages to the child
- `child.go`
  - initializes the Agora SDK
  - joins the channel
  - creates local tracks
  - consumes IPC messages
  - publishes audio and video

## Current Working Path

- Go wrapper: `v2.4.12` (`go-publish-video/go.mod`)
- raw input file: `I420` (YUV420), e.g. `test_data/send_video_cif.yuv`
- published pixel format: `I420` (`VideoPixelI420`, `VideoPublishTypeYuv`)
- no colour conversion happens anywhere; parent passes the bytes through

The `RGB24` -> `RGBA` conversion this document used to describe belongs to the
`linux_rtc_4_4_31` branch and does not exist on master.

## Parent Responsibilities

- own frame pacing
- calculate bytes-per-frame
- restart file reads at EOF
- preserve timestamps across IPC
- pass `-enableStringUID` correctly as a single flag argument

## Child Responsibilities

- own SDK state and lifecycle
- own connection and local tracks
- configure encoder settings
- publish samples with the timestamps received from the parent

## Known Fixes Already In This Branch

- bool flag passing for `enableStringUID`
- timestamp propagation from parent to child
- `I420` sample support

## Where To Debug

- format conversion bugs:
  - `go-publish-video/parent.go`
- track creation or publish failures:
  - `go-publish-video/child.go`
- structured message issues:
  - `go-publish-video/child_ipc.go`
- message shape changes:
  - `go-publish-video/ipc/ipc_defs.fbs`
