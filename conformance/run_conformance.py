#!/usr/bin/env python3
"""Conformance tests for an avatar implementation of the generic protocol.

Asserts what the avatar *plays*, not what it receives. See README.md.
"""
import argparse, array, asyncio, base64, json, subprocess, sys, time, uuid
from pathlib import Path

from tones import RATE, tone, present_ms
from scenarios import SCENARIOS, A_HZ, B_HZ, C_HZ


class Recording:
    """What the implementation published, queried by tone."""

    def __init__(self, pcm: bytes):
        self.pcm = pcm

    def present(self, hz):
        return present_ms(self.pcm, hz)

    def last(self, hz):
        """Millisecond at which this tone was last heard."""
        import array as _a
        a = _a.array("h"); a.frombytes(self.pcm[: len(self.pcm) // 2 * 2])
        step = int(RATE * 0.1)
        from tones import goertzel
        last = None
        for i in range(0, len(a) - step, step):
            if goertzel(a[i:i + step], hz) > 40.0:
                last = int((i + step) * 1000 / RATE)
        return last


async def drive(base_url, api_key, channel, frames, agora):
    """Start a session, then send the scenario's frames with their timing."""
    import requests, websockets

    body = {
        "avatar_id": agora["avatar_id"], "quality": "high", "version": "v1",
        "video_encoding": "H264", "activity_idle_timeout": 120, "area": "GLOBAL",
        "agora_settings": {
            "app_id": agora["app_id"], "token": agora["token"],
            "channel": channel, "uid": agora["uid"], "enable_string_uid": False,
        },
    }
    r = requests.post(f"{base_url.rstrip('/')}/session/start", json=body,
                      headers={"x-api-key": api_key}, timeout=30)
    r.raise_for_status()
    s = r.json()
    ws_url, token = s["websocket_address"], s["session_token"]
    seg = {}

    async with websockets.connect(ws_url, extra_headers={"authorization": f"Bearer {token}"}) as ws:
        await ws.send(json.dumps(dict(body, command="init", session_id=s["session_id"])))
        for delay_ms, frame in frames:
            if delay_ms:
                await asyncio.sleep(delay_ms / 1000)
            f = dict(frame)
            if "_tone" in f:
                hz, ms = f.pop("_tone")
                f["audio"] = base64.b64encode(tone(hz, ms)).decode()
                f["sampleRate"] = RATE
                f["encoding"] = "PCM16"
            if "segment_id" in f:
                f["segment_id"] = seg.setdefault(f["segment_id"], uuid.uuid4().hex)
            f["event_id"] = uuid.uuid4().hex
            await ws.send(json.dumps(f))
    return s["session_id"]


def report(name, results):
    ok = True
    print(f"\n{name}")
    for label, passed, detail in results:
        print(f"  {'PASS' if passed else 'FAIL'}  {label}  ({detail})")
        ok &= bool(passed)
    return ok


# ---------------------------------------------------------------- self-check

def _faults():
    """One deliberately broken recording per contract, with the assertion each must trip."""
    silence = lambda ms: b"\x00\x00" * int(RATE * ms / 1000)
    return {
        "truncated-by-next-segment": (
            "queueing", "segment A played in full",
            tone(A_HZ, 1000) + tone(B_HZ, 2000)),
        "second-segment-dropped": (
            "queueing", "segment B played in full",
            tone(A_HZ, 4000)),
        "segments-overlap": (
            "queueing", "B did not start before A finished",
            tone(B_HZ, 2000) + tone(A_HZ, 4000)),
        "interrupt-ignored": (
            "interrupt-stops", "segment A was cut short by the interrupt",
            tone(A_HZ, 4000)),
        "queue-not-cleared-on-interrupt": (
            "interrupt-stops", "queued segment B never played",
            tone(A_HZ, 1500) + tone(B_HZ, 2000)),
        "cancelled-audio-returns": (
            "after-interrupt", "no cancelled audio came back",
            tone(A_HZ, 1500) + tone(C_HZ, 2000) + tone(A_HZ, 2500)),
        "unknown-field-is-fatal": (
            "unknown-fields", "playback unaffected by an unknown field",
            silence(2000)),
    }


def self_check():
    print("Self-check: every fault must fail, and fail for its own reason.\n")
    ok = True
    for fault, (scenario, expect_label, pcm) in _faults().items():
        check = SCENARIOS[scenario]["check"]
        results = list(check(Recording(pcm)))
        failed = [l for l, passed, _ in results if not passed]
        hit = expect_label in failed
        only = failed == [expect_label]
        status = "ok" if hit else "NOT DETECTED"
        print(f"  {status:<13} {fault}")
        print(f"                 expected to trip: {expect_label!r}")
        if not hit:
            print(f"                 actually failed: {failed}")
        elif not only:
            print(f"                 (also tripped: {[f for f in failed if f != expect_label]})")
        ok &= hit
    print(f"\n{'SELF-CHECK PASSED' if ok else 'SELF-CHECK FAILED'}")
    return ok


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--base-url"); p.add_argument("--api-key")
    p.add_argument("--scenario", action="append")
    p.add_argument("--capture-cmd",
                   help="writes PCM16 mono 16k to {out} for {ms} from channel {channel}")
    p.add_argument("--avatar-id", default="conformance")
    p.add_argument("--list", action="store_true")
    p.add_argument("--self-check", action="store_true")
    a = p.parse_args()

    if a.list:
        for k, v in SCENARIOS.items():
            print(f"  {k:<18} {v['doc']}")
        return 0
    if a.self_check:
        return 0 if self_check() else 1
    if not (a.base_url and a.api_key and a.capture_cmd):
        p.error("--base-url, --api-key and --capture-cmd are required to run against an endpoint")
    print("Driving an endpoint requires Agora credentials; see README.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
