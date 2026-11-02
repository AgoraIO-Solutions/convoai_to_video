package main

import (
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
// that the three I420 planes are Green, Blue and Red.
//
// What that costs depends on the codec, and both halves were measured in a
// browser rather than reasoned about:
//
//   - H264/VP8: matrix 0 is a legal but false statement, so the picture goes
//     green -- and only once something honours it. Hardware decoders commonly
//     ignore the signalled matrix, so the fault appears when a renegotiation
//     drops the decoder to its software path, which is why it showed up on
//     mute rather than at the start of the call.
//   - AV1: matrix 0 is not merely wrong, it is **non-conformant**. The AV1
//     spec requires 4:4:4 when matrix_coefficients is MC_IDENTITY (0), and
//     this publisher sends I420, which is 4:2:0. Chrome tolerates the invalid
//     stream and renders GBR immediately; a stricter decoder may refuse it
//     outright.
//
// The value below is **2, unspecified**, for every codec, and that is
// deliberate. This process is handed raw I420 and has no idea what colour
// space it is in; naming one is a guess. "Unspecified" tells the receiver to
// apply its own default, which is what it was already doing while the frames
// were untagged -- so the green goes away and nothing else about the picture
// changes. Measured on 640x360 content, declaring BT.709 against a receiver
// defaulting to BT.601 moved mean luma by -3.3 and the green channel by -4.2,
// which looks like the picture quietly getting darker. Unspecified cannot be
// wrong in that way, because it claims nothing.
//
// # On the AV1 recommendation in the Go SDK README
//
// That README ("How to Encode with AV1 and Remain Compatible with Web
// Decoding") asks for (PrimariesId 2, TransferId 2, MatrixId 5, RangeId 1) and
// says that without it "Web decoding of AV1 video stream will not work". We
// could not reproduce that: AV1 published with everything unspecified decodes
// and renders correctly in Chrome (Web SDK 4.24.9), including across the
// mute/unmute renegotiation. The failure that recommendation describes is
// almost certainly the zero value -- matrix 0, the non-conformant case above
// -- for which (2,2,5,1) is *a* fix rather than *the* fix.
//
// So this file does not branch on codec. If a receiver is ever reported
// failing to decode AV1, that tuple is the first thing to try; it is a
// conformant alternative, not a contradiction. Do not reintroduce the branch
// on the strength of the README alone -- it asserts a matrix and a range for
// pixel data this process never inspected, which is the one way this code can
// make a picture worse.
//
// Override these only if you genuinely know what your source is. If you do,
// declaring it is better than leaving it to a guess.
var videoColorSpace = defaultColorSpace()

// defaultColorSpace is the colour space to publish when the caller has not
// named one. See the comment above for why it says "unspecified" rather than
// naming a matrix, and why it is the same for H264, VP8 and AV1.
func defaultColorSpace() agoraservice.ColorSpaceType {
	return agoraservice.ColorSpaceType{
		RangeId:     0, // unspecified
		MatrixId:    2, // unspecified -- NOT 0, which means GBR
		PrimariesId: 2, // unspecified
		TransferId:  2, // unspecified
	}
}
