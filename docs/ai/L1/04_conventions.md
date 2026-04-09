# 04 Conventions

> Code and repository conventions that matter when making changes here.

## General

- keep secrets out of source files
- prefer explicit CLI flags over hidden defaults for test runs
- preserve the split between protocol examples and publisher runtime code

## Documentation Conventions

- keep `AGENTS.md` as the root entrypoint for repo-local agent instructions
- keep `CLAUDE.md` as a thin redirect to `AGENTS.md`
- keep L1 summaries small enough to load together
- put longer subsystem docs under `docs/ai/L1/deep_dives/`

## Go Publisher

- parent and child communicate only through FlatBuffers IPC
- child owns all Agora SDK objects
- parent is responsible for file-loop timing
- log traffic from child returns over:
  - stderr for human-readable runtime logs
  - stdout IPC for structured status/log messages

## Code Style Preferences

- prefer explicit flags over hidden behavior
- keep media format conversions close to file reading code
- keep SDK interactions centralized in the child process
- avoid embedding environment-specific hacks outside the workaround branch

## Commit Conventions

- commit messages start with lowercase
- keep messages in present tense
- do not mention AI tools in commit messages
- do not add Co-Authored-By trailers
- let hooks run normally

## Branch Conventions

- current customer workaround branch: `linux_rtc_4_4_31`
- do not silently retarget that branch to a newer SDK line
- if testing a newer SDK, make that an explicit branch or isolated patch

## Media Conventions

- audio sample format: PCM16 mono by default
- workaround video input format: `RGB24`
- SDK publish format on workaround branch: `RGBA`
- timestamps should be propagated from parent to child

## Conversion Conventions

- file input is packed `RGB24`
- IPC payload carries packed `RGBA`
- width, height, and bytes-per-pixel must stay in sync
- frame timing should come from the parent sample loop, not child wall clock

## UID Conventions

- `-enableStringUID=true|false` must be passed as a single CLI arg
- all channel participants must use the same UID mode
- browser/web demo testing is simplest with `-enableStringUID=false`

## Testing

- use `make test` for Go unit tests
- prefer numeric UIDs for manual browser verification
- do not assume the Python mocks are production services

## What Not To Do

- do not upgrade the SDK line on the workaround branch without Modal verification
- do not mix numeric and string UID viewers in the same manual test
- do not treat the Python mocks as production-ready auth or persistence code
- do not let the child infer frame geometry from file size alone

## Related Deep Dives

- [Go Publisher Pipeline](deep_dives/go_publisher_pipeline.md) — timestamp and format handling details
- [Modal SDK Compatibility](deep_dives/modal_sdk_compatibility.md) — why the workaround branch stays pinned
