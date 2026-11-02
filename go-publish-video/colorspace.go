package main

import (
	"strings"

	agoraservice "github.com/AgoraIO-Extensions/Agora-Golang-Server-SDK/v2/go_sdk/rtc"
)

// Colour space sent with every video frame.
//
// It has to be set explicitly. `ExternalVideoFrame.ColorSpace` is a struct, so
// leaving it out does not mean "unspecified" -- Go zero-values it and the SDK
// copies all four fields to the native frame unconditionally. Its own source
// says so:
//
//	// fill color space: for go,the default value of struct 's member is zero value!
//	cFrame.color_space.matrix_id = C.int(frame.ColorSpace.MatrixId)
//
// These are H.273 indices, and in H.273 Table 4 the matrix index **0 is GBR**,
// not "unspecified" (which is 2). So an unset ColorSpace tells every receiver
// that the three I420 planes are Green, Blue and Red. A receiver that believes
// it reads the luma plane as green. Hardware decoders commonly ignore the
// signalled matrix, so the fault only shows where it is honoured -- which
// includes the software fallback a decoder rebuild drops to after a
// renegotiation. That is why it appears on mute and not before.
//
// The default is **2, unspecified** rather than a real matrix, and that is
// deliberate. This process is handed raw I420 and has no idea what colour
// space it is in; naming one is a guess. "Unspecified" tells the receiver to
// apply its own default, which is what it was already doing while the frames
// were untagged -- so the green goes away and nothing else about the picture
// changes.
//
// Naming a matrix instead is a visible change when the guess disagrees with
// the receiver's default, which is resolution-dependent: receivers assume
// BT.601 for standard-definition frames and BT.709 for high-definition ones.
// Measured here on 640x360 content, declaring BT.709 against a receiver
// defaulting to BT.601 moved mean luma by -3.3 and the green channel by -4.2,
// which looks like the picture quietly getting darker.
//
// **AV1 is the exception, and it is not optional.** The Go SDK's own README
// ("How to Encode with AV1 and Remain Compatible with Web Decoding") requires
// a real matrix there -- RangeId 1, MatrixId 5, PrimariesId 2, TransferId 2 --
// and says that without it "Web decoding of AV1 video stream will not work".
// That is a harder failure than a colour shift: the browser does not decode at
// all. So the default is chosen by codec below. (The same section also warns
// that for AV1 a non-nil MetadataBuffer carrying SEI stops the web client
// decoding, which is why this publisher never sets one.)
//
// So: override these only if you genuinely know what your source is. If you
// do, declaring it is better than leaving it to a guess.
var videoColorSpace = defaultColorSpaceFor("")

// defaultColorSpaceFor picks the colour space to publish when the caller has
// not named one. See the comment above for why AV1 differs.
func defaultColorSpaceFor(codec string) agoraservice.ColorSpaceType {
	if strings.EqualFold(codec, "AV1") {
		return agoraservice.ColorSpaceType{
			RangeId:     1, // limited / broadcast
			MatrixId:    5, // BT.470BG -- required for web decoding of AV1
			PrimariesId: 2, // unspecified
			TransferId:  2, // unspecified
		}
	}
	return agoraservice.ColorSpaceType{
		MatrixId:    2, // unspecified -- receiver's default, NOT 0, which means GBR
		RangeId:     0, // unspecified
		PrimariesId: 2, // unspecified
		TransferId:  2, // unspecified
	}
}

// codecFromArgs reads -videoCodec out of the raw arguments. The colour-space
// flag defaults depend on the codec, and a flag's default has to be fixed
// before flag.Parse runs, so this cannot wait for the parsed value.
func codecFromArgs(args []string) string {
	for i, a := range args {
		if a == "-videoCodec" || a == "--videoCodec" {
			if i+1 < len(args) {
				return args[i+1]
			}
			return ""
		}
		for _, pfx := range []string{"-videoCodec=", "--videoCodec="} {
			if strings.HasPrefix(a, pfx) {
				return strings.TrimPrefix(a, pfx)
			}
		}
	}
	return ""
}
