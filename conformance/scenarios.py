"""Scenarios, as data. Each fails for its own reason; see README.

A scenario is a list of (delay_ms, frame) and a check over the recording.
Frames use a placeholder segment id that the driver substitutes.
"""
from tones import tone

A_HZ, B_HZ, C_HZ = 440, 1200, 800


def _voice(seg, hz, ms):
    return {"command": "voice", "_tone": (hz, ms), "segment_id": seg}


def _end(seg):
    return {"command": "voice_end", "segment_id": seg}


def queueing(r):
    """Both segments play in full, in order. Covers queue, append, no truncation."""
    a, _ = r.present(A_HZ)
    b, b_start = r.present(B_HZ)
    a_end = r.last(A_HZ)
    yield "segment A played in full", a >= 3600, f"{a}ms of 4000ms"
    yield "segment B played in full", b >= 1700, f"{b}ms of 2000ms"
    yield "B did not start before A finished", (b_start or 0) >= (a_end or 0) - 300, \
          f"B at {b_start}ms, A ended {a_end}ms"


def interrupt_stops(r):
    """voice_interrupt stops what plays AND clears what is queued."""
    a, _ = r.present(A_HZ)
    b, _ = r.present(B_HZ)
    yield "segment A was cut short by the interrupt", a <= 2400, f"{a}ms, expected <=2400"
    yield "segment A did play before it", a >= 600, f"{a}ms, expected >=600"
    yield "queued segment B never played", b <= 200, f"{b}ms, expected ~0"


def after_interrupt(r):
    """A segment sent after an interrupt plays, and cancelled audio stays gone."""
    c, _ = r.present(C_HZ)
    a, _ = r.present(A_HZ)
    yield "the replacement segment played in full", c >= 1700, f"{c}ms of 2000ms"
    yield "no cancelled audio came back", a <= 2400, f"{a}ms of the interrupted segment"


def unknown_fields(r):
    """An unrecognised top-level field is ignored, not fatal."""
    a, _ = r.present(A_HZ)
    yield "playback unaffected by an unknown field", a >= 1700, f"{a}ms of 2000ms"


SCENARIOS = {
    "queueing": {
        "doc": "two segments back to back both play in full, in order",
        "frames": [
            (0,    _voice("A", A_HZ, 2000)),
            (0,    _voice("A", A_HZ, 2000)),
            (0,    _end("A")),
            (1000, _voice("B", B_HZ, 2000)),   # arrives while A is still playing
            (0,    _end("B")),
        ],
        "record_ms": 9000,
        "check": queueing,
    },
    "interrupt-stops": {
        "doc": "voice_interrupt stops the playing segment and clears the queue",
        "frames": [
            (0,    _voice("A", A_HZ, 4000)),
            (0,    _end("A")),
            (200,  _voice("B", B_HZ, 2000)),
            (0,    _end("B")),
            (1500, {"command": "voice_interrupt"}),
        ],
        "record_ms": 9000,
        "check": interrupt_stops,
    },
    "after-interrupt": {
        "doc": "a segment sent after an interrupt plays, cancelled audio does not return",
        "frames": [
            (0,    _voice("A", A_HZ, 4000)),
            (0,    _end("A")),
            (1500, {"command": "voice_interrupt"}),
            (500,  _voice("C", C_HZ, 2000)),
            (0,    _end("C")),
        ],
        "record_ms": 10000,
        "check": after_interrupt,
    },
    "unknown-fields": {
        "doc": "an unrecognised top-level field is ignored, not fatal",
        "frames": [
            (0, dict(_voice("A", A_HZ, 2000), some_future_field={"a": 1})),
            (0, _end("A")),
        ],
        "record_ms": 6000,
        "check": unknown_fields,
    },
}
