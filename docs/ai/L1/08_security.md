# 08 Security

> Trust boundaries, secret handling, and the main security assumptions in this repo.

## Secrets

- never hardcode App IDs, tokens, or certificates into source
- pass credentials by flags or environment variables
- token generation belongs in test/dev tooling, not the main publisher loop

## Repo-Specific Secret Locations

- `go-publish-video/` flags accept publish credentials at runtime
- Python mock clients may send API keys or bearer tokens in local tests
- logs and shell history can leak tokens if commands are pasted carelessly

## Trust Boundaries

- Agora calls the avatar provider REST and WebSocket endpoints
- the publisher process uses Agora credentials to join a channel
- media files under `test_data/` are trusted local test assets

## Input Validation

- session setup endpoints should validate API key or caller auth
- WebSocket receiver should validate bearer token before accepting audio
- raw media files used for local testing are trusted, but external media inputs should be validated before use

## UID / Token Safety

- token type must match UID mode
- string and numeric UID channels should not be mixed

## Logging

- avoid printing full tokens in stdout, stderr, or SDK log files
- keep generated SDK logs out of permanent source control history
- treat local test logs as internal artifacts, not docs

## External Interfaces

- REST and WebSocket examples are protocol-facing code
- validate auth and tokens before accepting session or audio traffic

## Operational Safety

- do not expose the mock receivers directly to the public internet as production services
- do not rely on in-memory session stores for real production lifecycle control
- isolate customer workaround branches from experimental SDK upgrades

## Token Handling Practices

- prefer short-lived publish tokens for manual testing
- treat copied web-demo join links as sensitive when they embed credentials
- rotate any credential that was printed into logs during debugging

## Dependency Safety

- the local SDK checkout is part of the trusted build chain for the Go publisher
- verify the SDK version before copying `agora_sdk/` into the repo working directory
- do not mix artifacts from multiple SDK versions in the same `go-publish-video/agora_sdk/` folder

## Reviewer Red Flags

- committed App IDs, certificates, or long-lived tokens
- test assets or logs that unexpectedly contain customer data
- mock endpoints being described as production-ready flows

## Safer Review Checklist

- credentials passed at runtime, not committed
- viewer UID mode matches publish mode
- tests do not contain real secrets
- SDK logs are not accidentally staged
- mock auth remains clearly labeled as example-only
- workaround branch docs still describe the actual live SDK line

## Related Deep Dives

- [ConvoAI Protocol Notes](deep_dives/convoai_protocol.md) — auth-related boundaries in the mock services
- [Modal SDK Compatibility](deep_dives/modal_sdk_compatibility.md) — environment-specific operational caution
