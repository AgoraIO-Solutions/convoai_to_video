# Protocol conformance tests

Checks an avatar implementation against
[`websocket-receive-audio/README.md`](../websocket-receive-audio/README.md).
Run it against your own endpoint before going live.

The suite asserts **what the avatar plays**, not what it receives. Taking a
frame is not the same as acting on it: an implementation can accept two speech
segments perfectly and still cut the first one off. That is the failure this
exists to catch, and a receipt-based check would pass it.

## How it works

Each segment is a distinct tone, so the published audio says which segments
played and for how long.

1. `POST /session/start` against your endpoint
2. connect to the WebSocket it returns, send `init`
3. send the scenario's frames — tones as `voice`, closed by `voice_end`
4. record the Agora channel your implementation publishes into
5. assert on the recording: which tones are present, for how long, in what order

## Running

```bash
pip install -r requirements.txt
export AGORA_APP_ID=... AGORA_APP_CERTIFICATE=...
python run_conformance.py --base-url https://your.host/api --api-key YOUR_KEY
```

Add `--scenario queueing` to run one. `--list` shows them all.

## What each scenario proves

| Scenario | Passes only if |
| --- | --- |
| `queueing` | two segments sent back to back **both play in full**, in order |
| `no-truncation` | a `voice` arriving mid-playback appends; the first tone is not cut short |
| `interrupt-stops` | `voice_interrupt` stops the playing segment **and** clears what is queued |
| `interrupt-only` | nothing else truncates: no `voice_end`, no new `voice`, no `special` |
| `after-interrupt` | a segment sent after an interrupt plays in full, and no cancelled audio returns |
| `gesture-bound` | a `special` carrying a `segment_id` acts when **that** segment begins |
| `gesture-immediate` | a `special` with no `segment_id` acts at once, even with speech queued |
| `unknown-fields` | an unrecognised top-level field is ignored, not fatal |

Gesture scenarios need your implementation to report when it acted. If it
cannot, they are skipped rather than passed — see `--gesture-hook`.

## Self-check

The suite ships deliberately broken captures under `fixtures/faults/`, one per
contract: a segment id missing, an id changed mid-utterance, an id reused
across utterances, a gesture bound to the wrong segment, and an interrupt that
should not be there. `python run_conformance.py --self-check` asserts each one
**fails for its own reason**.

That exists because a suite is only as good as its ability to fail. If you
weaken an assertion to make your implementation pass, the matching fault stops
being detected and the self-check fails.

Note the limit honestly: nothing stops someone editing a fixture and its
expectation together. Treat changes under `fixtures/` as needing review.
