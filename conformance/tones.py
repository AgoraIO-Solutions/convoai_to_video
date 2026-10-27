"""Distinct tones per segment, so a recording says what played and for how long."""
import math, struct

RATE = 16000


def tone(freq_hz: int, ms: int, amplitude: int = 9000) -> bytes:
    """PCM16 mono at RATE."""
    n = int(RATE * ms / 1000)
    return b"".join(
        struct.pack("<h", int(amplitude * math.sin(2 * math.pi * freq_hz * i / RATE)))
        for i in range(n)
    )


def goertzel(samples, freq_hz: int) -> float:
    """Energy at one frequency. Cheaper and clearer here than a full FFT."""
    n = len(samples)
    if n == 0:
        return 0.0
    k = int(0.5 + n * freq_hz / RATE)
    w = 2 * math.pi * k / n
    cw, sw = math.cos(w), math.sin(w)
    coeff = 2 * cw
    s0 = s1 = s2 = 0.0
    for x in samples:
        s0 = x + coeff * s1 - s2
        s2, s1 = s1, s0
    return math.sqrt(s1 * s1 + s2 * s2 - coeff * s1 * s2) / n


def present_ms(pcm: bytes, freq_hz: int, window_ms: int = 100, floor: float = 40.0):
    """Total milliseconds in which `freq_hz` dominates, and when it first appears."""
    import array
    a = array.array("h")
    a.frombytes(pcm[: len(pcm) // 2 * 2])
    step = int(RATE * window_ms / 1000)
    total, first = 0, None
    for i in range(0, len(a) - step, step):
        win = a[i : i + step]
        if goertzel(win, freq_hz) > floor:
            total += window_ms
            if first is None:
                first = int(i * 1000 / RATE)
    return total, first
