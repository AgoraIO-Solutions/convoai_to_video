package main

import "testing"

func TestDefaultColorSpaceSaysUnspecified(t *testing.T) {
	// The whole bug is matrix 0. H.273 index 0 is GBR, and the Go zero value
	// of the ColorSpace struct is exactly that -- so "we never set it" and "we
	// told the receiver the planes are Green, Blue and Red" are the same
	// statement. Nothing else in this test matters as much as matrix != 0.
	cs := defaultColorSpace()
	if cs.MatrixId == 0 {
		t.Fatalf("matrix 0 means GBR, and on AV1 is not even conformant for 4:2:0: %+v", cs)
	}
	if cs.MatrixId != 2 || cs.RangeId != 0 || cs.PrimariesId != 2 || cs.TransferId != 2 {
		t.Fatalf("default = %+v, want matrix 2 / range 0 / primaries 2 / transfer 2", cs)
	}
}
