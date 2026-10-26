# Modal SDK Compatibility

## Purpose

Record the tested Linux SDK compatibility boundary for Modal/gVisor.

**Scope: the `linux_rtc_4_4_31` branch.** Master tracks Go SDK `v2.4.12`; the
versions below are that branch's, kept so the Modal finding is not lost.

## Tested Boundary

- known-good:
  - Linux RTC `4.4.31`
- known-bad:
  - Linux RTC `4.4.32`

## What Failed

- newer Linux RTC `4.4.32` failed during very early startup on Modal/gVisor
- the direct native sample path never reached connect/join/publish
- the same failure pattern also showed up through Go and Python server SDK wrappers because they sit on top of the native SDK line

## Practical Repo Rule

- keep the `linux_rtc_4_4_31` branch pinned to the `4.4.31` line
- do not upgrade the native SDK on this branch without repeating a real Modal test

## Verification Expectations

- if changing the publish path, confirm:
  - channel join works
  - audio renders
  - video renders
  - UID mode matches the chosen viewer

## Related Files

- `go-publish-video/README.md`
- `go-publish-video/go.mod`
- `docs/ai/L1/07_gotchas.md`
