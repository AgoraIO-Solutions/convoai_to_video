# ConvoAI To Video Repo Card

> Repo identity, quick map, and L1 index for Agora ConvoAI avatar-provider integration.

## Identity

- Purpose: reference integration for receiving ConvoAI speech, generating avatar video, and publishing audio/video back into Agora
- Main languages: Python and Go
- Primary runtime pieces:
  - `connection-setup/` mock REST session API
  - `websocket-receive-audio/` mock WebSocket audio receiver
  - `go-publish-video/` Agora publisher
- Current workaround branch focus: `go-publish-video/` is pinned to a Modal-compatible `4.4.31` SDK path and publishes `RGB24` input packed to `RGBA`

## When To Read What

- Session setup / REST questions: read `L1/06_interfaces.md`
- Local build and run: read `L1/01_setup.md`
- Where code lives: read `L1/03_code_map.md`
- Common publisher edits: read `L1/05_workflows.md`
- Modal / SDK caveats: read `L1/07_gotchas.md`

## L1 Index

- [L1/01_setup.md](L1/01_setup.md)
- [L1/02_architecture.md](L1/02_architecture.md)
- [L1/03_code_map.md](L1/03_code_map.md)
- [L1/04_conventions.md](L1/04_conventions.md)
- [L1/05_workflows.md](L1/05_workflows.md)
- [L1/06_interfaces.md](L1/06_interfaces.md)
- [L1/07_gotchas.md](L1/07_gotchas.md)
- [L1/08_security.md](L1/08_security.md)

