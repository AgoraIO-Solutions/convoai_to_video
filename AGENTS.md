# AI Agent Instructions

This repository uses progressive disclosure documentation. Docs live under
`docs/ai/` in three levels.

## How to Load

1. Read [docs/ai/L0_repo_card.md](docs/ai/L0_repo_card.md) to identify the repo.
2. Load ALL 8 files in `docs/ai/L1/`. They are small and intended to load together.
3. Follow L2 deep-dive links only when L1 is not detailed enough.

## Git Conventions

- **Lowercase start**: commit messages begin with a lowercase letter
- **No AI tool names**: do not mention AI tools in commit messages or trailers
- **Present tense**: use present-tense commit messages
- **No Co-Authored-By trailers**: omit AI attribution lines
- **No --no-verify**: let hooks run normally
- **No git config changes**: do not modify `user.name` or `user.email`

## Doc Commands

| Command | When to use |
| --- | --- |
| `generate docs` | `docs/ai/` does not exist yet |
| `update docs` | code or behavior changed |
| `test docs` | verify the docs are sufficient for common agent tasks |

Canonical standard:
- [progressive-disclosure-standard.md](docs/progressive-disclosure-standard.md)

