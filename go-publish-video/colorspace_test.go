package main

import "testing"

func TestDefaultColorSpaceByCodec(t *testing.T) {
	// AV1: the SDK requires a real matrix or the web client cannot decode.
	av1 := defaultColorSpaceFor("AV1")
	if av1.RangeId != 1 || av1.MatrixId != 5 || av1.PrimariesId != 2 || av1.TransferId != 2 {
		t.Fatalf("AV1 default = %+v, want {2 2 5 1} in (primaries, transfer, matrix, range)", av1)
	}
	if lower := defaultColorSpaceFor("av1"); lower != av1 {
		t.Fatalf("codec match must be case-insensitive: %+v", lower)
	}
	// Everything else: say nothing, so the receiver keeps its own default.
	for _, c := range []string{"H264", "VP8", ""} {
		got := defaultColorSpaceFor(c)
		if got.MatrixId != 2 || got.RangeId != 0 {
			t.Fatalf("%s default = %+v, want matrix 2 / range 0", c, got)
		}
		if got.MatrixId == 0 {
			t.Fatalf("%s: matrix 0 means GBR, never a default", c)
		}
	}
}

func TestCodecFromArgs(t *testing.T) {
	cases := map[string][]string{
		"AV1":  {"-appID", "x", "-videoCodec", "AV1"},
		"H264": {"--videoCodec=H264"},
		"":     {"-appID", "x"},
	}
	for want, args := range cases {
		if got := codecFromArgs(args); got != want {
			t.Fatalf("codecFromArgs(%v) = %q, want %q", args, got, want)
		}
	}
	if got := codecFromArgs([]string{"-videoCodec"}); got != "" {
		t.Fatalf("trailing -videoCodec with no value = %q, want empty", got)
	}
}
