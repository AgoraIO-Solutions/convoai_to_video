# Go Publisher Pipeline

## Purpose

Deep dive for the `go-publish-video/` process split, media loop, and the current Modal workaround path.

## Process Model

- `parent.go`
  - parses flags
  - opens media files
  - loops sample audio/video
  - converts `RGB24` to `RGBA`
  - writes FlatBuffers IPC messages to the child
- `child.go`
  - initializes the Agora SDK
  - joins the channel
  - creates local tracks
  - consumes IPC messages
  - publishes audio and video

## Current Working Path

- branch: `linux_rtc_4_4_31`
- Go wrapper: `v2.2.4`
- native Linux RTC line: `4.4.31`
- raw input file: `RGB24`
- published pixel format: `RGBA`

## Why RGB24 -> RGBA

- the customer workaround branch avoids the earlier `YUV`-oriented sample path
- parent-side conversion is explicit and easy to inspect
- `RGBA` is the stable SDK input used on the verified branch

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
- RGB24 sample support and conversion to RGBA

## Where To Debug

- format conversion bugs:
  - `go-publish-video/parent.go`
- track creation or publish failures:
  - `go-publish-video/child.go`
- structured message issues:
  - `go-publish-video/child_ipc.go`
- message shape changes:
  - `go-publish-video/ipc/ipc_defs.fbs`
