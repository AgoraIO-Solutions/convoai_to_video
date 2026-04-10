//go:build !test

package main

import (
	"bufio"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"go-publish-video/ipc/ipcgen"

	agoraservice "github.com/AgoraIO-Extensions/Agora-Golang-Server-SDK/v2/go_sdk/rtc"
)

var (
	// Global Agora SDK objects
	rtcConnection     *agoraservice.RtcConnection
	initWidth         int32
	initHeight        int32
	initFrameRate     int32
	initVideoCodec    agoraservice.VideoCodecType
	initSampleRate    int32
	initAudioChannels int32
	initBitrate       int
	initMinBitrate    int

	globalAppID   string
	globalChannel string
	globalUserID  string
	globalCodecName string
)

func onConnected(conn *agoraservice.RtcConnection, conInfo *agoraservice.RtcConnectionInfo, reason int) {
	logMsg := fmt.Sprintf("Agora SDK: Connected. UserID: %s, Channel: %s, Reason: %d", conInfo.LocalUserId, conInfo.ChannelId, reason)
	childLogger.Println(logMsg)
	sendAsyncLogResponse(ipcgen.LogLevelINFO, logMsg)

	if err := setupMediaInfrastructureAndPublish(conn); err != nil {
		errMsg := fmt.Sprintf("Failed to setup media infrastructure: %v", err)
		childLogger.Println("ERROR: " + errMsg)
		sendAsyncErrorResponse(ipcgen.ConnectionStatusFAILED, errMsg, "MediaSetupError")
	} else {
		successMsg := fmt.Sprintf("Successfully connected and media infrastructure prepared. Codec: %s", globalCodecName)
		sendAsyncStatusResponse(ipcgen.ConnectionStatusCONNECTED, successMsg, "")
	}
}

func onDisconnected(conn *agoraservice.RtcConnection, conInfo *agoraservice.RtcConnectionInfo, reason int) {
	logMsg := fmt.Sprintf("Agora SDK: Disconnected. Reason: %d", reason)
	childLogger.Println(logMsg)
	sendAsyncLogResponse(ipcgen.LogLevelWARN, logMsg)
	sendAsyncStatusResponse(ipcgen.ConnectionStatusDISCONNECTED, logMsg, "")
}

func onReconnecting(conn *agoraservice.RtcConnection, conInfo *agoraservice.RtcConnectionInfo, reason int) {
	logMsg := fmt.Sprintf("Agora SDK: Reconnecting... Reason: %d", reason)
	childLogger.Println(logMsg)
	sendAsyncLogResponse(ipcgen.LogLevelINFO, logMsg)
	sendAsyncStatusResponse(ipcgen.ConnectionStatusRECONNECTING, logMsg, "")
}

func onReconnected(conn *agoraservice.RtcConnection, conInfo *agoraservice.RtcConnectionInfo, reason int) {
	logMsg := fmt.Sprintf("Agora SDK: Reconnected. UserID: %s, Channel: %s, Reason: %d", conInfo.LocalUserId, conInfo.ChannelId, reason)
	childLogger.Println(logMsg)
	sendAsyncLogResponse(ipcgen.LogLevelINFO, logMsg)
	sendAsyncStatusResponse(ipcgen.ConnectionStatusRECONNECTED, "Successfully reconnected.", "")
}

func onConnectionLost(conn *agoraservice.RtcConnection, conInfo *agoraservice.RtcConnectionInfo) {
	logMsg := fmt.Sprintf("Agora SDK: Connection lost. UserID: %s, Channel: %s", conInfo.LocalUserId, conInfo.ChannelId)
	childLogger.Println("ERROR: " + logMsg)
	sendAsyncLogResponse(ipcgen.LogLevelERROR, logMsg)
	sendAsyncStatusResponse(ipcgen.ConnectionStatusCONNECTION_LOST, logMsg, "")
}

func onConnectionFailure(conn *agoraservice.RtcConnection, conInfo *agoraservice.RtcConnectionInfo, errCode int) {
	logMsg := fmt.Sprintf("Agora SDK: Connection failure. Error Code: %d", errCode)
	childLogger.Println("ERROR: " + logMsg)
	sendAsyncLogResponse(ipcgen.LogLevelERROR, logMsg)
	sendAsyncErrorResponse(ipcgen.ConnectionStatusFAILED, logMsg, fmt.Sprintf("AgoraErrorCode: %d", errCode))
}

func onUserJoined(conn *agoraservice.RtcConnection, uid string) {
	logMsg := fmt.Sprintf("Agora SDK: User %s joined", uid)
	childLogger.Println(logMsg)
	sendAsyncLogResponse(ipcgen.LogLevelINFO, logMsg)
}

func onUserLeft(conn *agoraservice.RtcConnection, uid string, reason int) {
	logMsg := fmt.Sprintf("Agora SDK: User %s left. Reason: %d", uid, reason)
	childLogger.Println(logMsg)
	sendAsyncLogResponse(ipcgen.LogLevelINFO, logMsg)
}

func onError(conn *agoraservice.RtcConnection, err int, msg string) {
	logMsg := fmt.Sprintf("Agora SDK: Error. Code: %d, Message: %s", err, msg)
	childLogger.Println("ERROR: " + logMsg)
	sendAsyncLogResponse(ipcgen.LogLevelERROR, logMsg)
}

func onTokenPrivilegeWillExpire(conn *agoraservice.RtcConnection, token string) {
	logMsg := "Agora SDK: Token privilege will expire soon. New token required."
	childLogger.Println("WARN: " + logMsg)
	sendAsyncLogResponse(ipcgen.LogLevelWARN, logMsg)
	sendAsyncStatusResponse(ipcgen.ConnectionStatusTOKEN_WILL_EXPIRE, "Token privilege will expire.", token)
}

func onTokenPrivilegeDidExpire(conn *agoraservice.RtcConnection) {
	logMsg := "Agora SDK: Token privilege did expire."
	childLogger.Println("WARN: " + logMsg)
	sendAsyncLogResponse(ipcgen.LogLevelWARN, logMsg)
	sendAsyncStatusResponse(ipcgen.ConnectionStatusFAILED, "Token privilege did expire.", "Token_Expired_Detail")
}

func cleanupLocalRtcResources(releaseConnectionObject bool) {
	childLogger.Println("Cleaning up local Agora RTC resources...")

	if rtcConnection != nil {
		// Unpublish streams
		rtcConnection.UnpublishAudio()
		rtcConnection.UnpublishVideo()

		if releaseConnectionObject {
			childLogger.Println("Disconnecting and Releasing RtcConnection object...")
			rtcConnection.Disconnect()
			rtcConnection.Release()
			rtcConnection = nil
		} else {
			childLogger.Println("Disconnecting RtcConnection (but not releasing object)...")
			rtcConnection.Disconnect()
		}
	}
	childLogger.Println("Local Agora RTC resources cleanup attempt finished.")
}

func main() {
	// Redirect stdout to /dev/null to prevent Agora SDK from polluting it
	// Save original stdout first
	originalStdout := os.Stdout
	devNull, _ := os.OpenFile("/dev/null", os.O_WRONLY, 0)
	os.Stdout = devNull

	// Set up logging to stderr
	childLogger = log.New(os.Stderr, "[agora_worker] ", log.LstdFlags|log.Lshortfile)
	childLogger.Println("Agora child process started.")

	// Use the original stdout for IPC communication
	stdoutWriter = bufio.NewWriter(originalStdout)

	// Define command-line flags
	appIDFlag := flag.String("appID", "", "Agora App ID")
	channelNameFlag := flag.String("channelName", "", "Agora Channel Name")
	userIDFlag := flag.String("userID", "", "Agora User ID for the child process")
	tokenFlag := flag.String("token", "", "Agora Token for the child process")
	widthFlag := flag.Int("width", 352, "Video width")
	heightFlag := flag.Int("height", 288, "Video height")
	frameRateFlag := flag.Int("frameRate", 15, "Video frame rate")
	videoCodecFlag := flag.String("videoCodec", "H264", "Video codec (H264, VP8, or AV1)")
	sampleRateFlag := flag.Int("sampleRate", 16000, "Audio sample rate")
	audioChannelsFlag := flag.Int("audioChannels", 1, "Audio channels")
	bitrateFlag := flag.Int("bitrate", 1000, "Video target bitrate in Kbps")
	minBitrateFlag := flag.Int("minBitrate", 100, "Video minimum bitrate in Kbps")
	enableStringUIDFlag := flag.Bool("enableStringUID", false, "Enable string UID support")

	flag.Parse()

	globalAppID = *appIDFlag
	globalChannel = *channelNameFlag
	globalUserID = *userIDFlag
	globalCodecName = *videoCodecFlag
	childProcessToken := *tokenFlag
	initWidth = int32(*widthFlag)
	initHeight = int32(*heightFlag)
	initFrameRate = int32(*frameRateFlag)
	initSampleRate = int32(*sampleRateFlag)
	initAudioChannels = int32(*audioChannelsFlag)
	initBitrate = *bitrateFlag
	initMinBitrate = *minBitrateFlag
	enableStringUID := *enableStringUIDFlag

	childLogger.Printf("Initial parameters from command line: AppID=%s, Channel=%s, UserID=%s, Codec=%s, Res=%dx%d@%d, Bitrate=%dKbps, MinBitrate=%dKbps, AudioSR=%d, AudioCh=%d, StringUID=%t",
		globalAppID, globalChannel, globalUserID, *videoCodecFlag, initWidth, initHeight, initFrameRate, initBitrate, initMinBitrate, initSampleRate, initAudioChannels, enableStringUID)

	// Add a small delay to ensure stdout redirection is complete
	time.Sleep(100 * time.Millisecond)

	serviceCfg := agoraservice.NewAgoraServiceConfig()
	serviceCfg.EnableAudioProcessor = true
	serviceCfg.EnableVideo = true
	serviceCfg.AppId = globalAppID
	serviceCfg.UseStringUid = enableStringUID
	serviceCfg.LogPath = "./agora_child_sdk.log"
	serviceCfg.LogSize = 5 * 1024 * 1024
	serviceCfg.LogLevel = 5  // Error only

	if ret := agoraservice.Initialize(serviceCfg); ret != 0 {
		errMsg := fmt.Sprintf("Agora SDK global Initialize() failed with code: %d", ret)
		childLogger.Println("FATAL: " + errMsg)
		sendErrorResponse(ipcgen.ConnectionStatusINITIALIZED_FAILURE, errMsg, "GlobalInitializeFailed")
		os.Exit(1)
	}
	childLogger.Println("Agora SDK global Initialize() successful.")
	defer agoraservice.Release()

	// Determine video codec type from flags
	switch *videoCodecFlag {
	case "H264":
		initVideoCodec = agoraservice.VideoCodecTypeH264
	case "VP8":
		initVideoCodec = agoraservice.VideoCodecTypeVp8
	case "AV1":
		initVideoCodec = agoraservice.VideoCodecTypeAv1
		// AV1 typically needs higher bitrates for real-time encoding
		if initBitrate < 1500 {
			childLogger.Printf("INFO: Adjusting bitrate from %d to 1500 Kbps for AV1 codec", initBitrate)
			initBitrate = 1500
		}
		if initMinBitrate < 500 {
			childLogger.Printf("INFO: Adjusting min bitrate from %d to 500 Kbps for AV1 codec", initMinBitrate)
			initMinBitrate = 500
		}
	default:
		childLogger.Printf("WARN: Unsupported video_codec_name '%s' from CLI, defaulting to H264 for Agora.", *videoCodecFlag)
		initVideoCodec = agoraservice.VideoCodecTypeH264
		globalCodecName = "H264"
	}

	childLogger.Printf("Selected codec: %s", globalCodecName)

	// Connection configuration
	connCfg := &agoraservice.RtcConnectionConfig{
		AutoSubscribeAudio: false,
		AutoSubscribeVideo: false,
		ClientRole:         agoraservice.ClientRoleBroadcaster,
		ChannelProfile:     agoraservice.ChannelProfileLiveBroadcasting,
	}

	// Publish configuration
	publishConfig := agoraservice.NewRtcConPublishConfig()
	publishConfig.AudioScenario = agoraservice.AudioScenarioDefault
	publishConfig.IsPublishAudio = true
	publishConfig.IsPublishVideo = true
	publishConfig.AudioProfile = agoraservice.AudioProfileDefault
	publishConfig.AudioPublishType = agoraservice.AudioPublishTypePcm
	publishConfig.VideoPublishType = agoraservice.VideoPublishTypeYuv

	rtcConnection = agoraservice.NewRtcConnection(connCfg, publishConfig)
	if rtcConnection == nil {
		errMsg := "Failed to create Agora RtcConnection instance."
		childLogger.Println("ERROR: " + errMsg)
		sendErrorResponse(ipcgen.ConnectionStatusINITIALIZED_FAILURE, errMsg, "NewRtcConnectionFailed")
		os.Exit(1)
	}

	observer := &agoraservice.RtcConnectionObserver{
		OnConnected:    onConnected,
		OnDisconnected: onDisconnected,
		OnConnecting: func(conn *agoraservice.RtcConnection, conInfo *agoraservice.RtcConnectionInfo, reason int) {
			logMsg := fmt.Sprintf("Agora SDK: Connecting... UserID: %s, Channel: %s, Reason: %d", conInfo.LocalUserId, conInfo.ChannelId, reason)
			childLogger.Println(logMsg)
			sendAsyncLogResponse(ipcgen.LogLevelINFO, "Connecting...")
		},
		OnReconnecting:             onReconnecting,
		OnReconnected:              onReconnected,
		OnConnectionLost:           onConnectionLost,
		OnConnectionFailure:        onConnectionFailure,
		OnTokenPrivilegeWillExpire: onTokenPrivilegeWillExpire,
		OnTokenPrivilegeDidExpire:  onTokenPrivilegeDidExpire,
		OnUserJoined:               onUserJoined,
		OnUserLeft:                 onUserLeft,
		OnError:                    onError,
	}

	rtcConnection.RegisterObserver(observer)
	childLogger.Println("Agora RtcConnection created and observer registered.")

	// Add delay before connect to let SDK finish initialization
	time.Sleep(200 * time.Millisecond)

	ret := rtcConnection.Connect(childProcessToken, globalChannel, globalUserID)
	if ret != 0 {
		errMsg := fmt.Sprintf("Agora RtcConnection.Connect() call failed with code: %d", ret)
		childLogger.Println("ERROR: " + errMsg)
		rtcConnection.Release()
		rtcConnection = nil
		sendErrorResponse(ipcgen.ConnectionStatusINITIALIZED_FAILURE, errMsg, "ConnectFailed")
		os.Exit(1)
	}
	childLogger.Printf("Agora RtcConnection.Connect() called for channel '%s', user '%s' with %s codec. Waiting for connection callbacks.",
		globalChannel, globalUserID, globalCodecName)

	// Add delay after connect to ensure no stdout pollution
	time.Sleep(100 * time.Millisecond)
	sendStatusResponse(ipcgen.ConnectionStatusINITIALIZED_SUCCESS, fmt.Sprintf("Connect call issued with %s codec, awaiting callback.", globalCodecName), "")

	reader := bufio.NewReader(os.Stdin)

	for {
		// Read 4-byte length prefix
		lenBytes := make([]byte, 4)
		if _, err := io.ReadFull(reader, lenBytes); err != nil {
			if err == io.EOF {
				childLogger.Println("Stdin closed, parent process likely terminated. Exiting.")
			} else {
				childLogger.Printf("Error reading message length from stdin: %v. Exiting.", err)
			}
			return
		}
		msgLen := binary.BigEndian.Uint32(lenBytes)

		if msgLen == 0 {
			childLogger.Println("Received 0-length message, skipping.")
			continue
		}

		// Read the message payload
		msgBuf := make([]byte, msgLen)
		if _, err := io.ReadFull(reader, msgBuf); err != nil {
			childLogger.Printf("Error reading message payload (len %d) from stdin: %v. Exiting.", msgLen, err)
			return
		}

		// Parse FlatBuffer message
		ipcMsg := ipcgen.GetRootAsIPCMessage(msgBuf, 0)

		// Get payload data as bytes
		payloadLen := ipcMsg.PayloadLength()
		if payloadLen == 0 && ipcMsg.MessageType() != ipcgen.MessageTypeCLOSE_COMMAND {
			childLogger.Printf("No payload for message type: %s", ipcgen.EnumNamesMessageType[ipcMsg.MessageType()])
			// Some messages like CLOSE_COMMAND don't have payload
			if ipcMsg.MessageType() == ipcgen.MessageTypeCLOSE_COMMAND {
				childLogger.Println("Received Close command. Cleaning up and exiting.")
				cleanupAgoraResources()
				sendAsyncLogResponse(ipcgen.LogLevelINFO, "Child process shutting down.")
				sendAsyncStatusResponse(ipcgen.ConnectionStatusDISCONNECTED, "", "Closed by parent command")
				childLogger.Println("Child process terminated by close command.")
				return
			}
			continue
		}

		// Extract payload bytes
		payloadBytes := make([]byte, payloadLen)
		for i := 0; i < payloadLen; i++ {
			payloadBytes[i] = byte(ipcMsg.Payload(i))
		}

		switch ipcMsg.MessageType() {
		case ipcgen.MessageTypeWRITE_VIDEO_SAMPLE_COMMAND:
			if rtcConnection == nil {
				continue
			}

			// Parse MediaSamplePayload from payload bytes
			samplePayload := ipcgen.GetRootAsMediaSamplePayload(payloadBytes, 0)
			dataLen := samplePayload.DataLength()
			if dataLen == 0 {
				continue
			}

			// Extract frame data
			frameData := make([]byte, dataLen)
			for i := 0; i < int(dataLen); i++ {
				frameData[i] = byte(samplePayload.Data(i))
			}

			extFrame := &agoraservice.ExternalVideoFrame{
				Type:      agoraservice.VideoBufferRawData,
				Format:    agoraservice.VideoPixelI420,
				Buffer:    frameData,
				Stride:    int(initWidth),
				Height:    int(initHeight),
				Timestamp: samplePayload.TimestampUnixNano() / 1_000_000,
			}
			rtcConnection.PushVideoFrame(extFrame)

		case ipcgen.MessageTypeWRITE_AUDIO_SAMPLE_COMMAND:
			if rtcConnection == nil {
				continue
			}

			// Parse MediaSamplePayload from payload bytes
			samplePayload := ipcgen.GetRootAsMediaSamplePayload(payloadBytes, 0)
			dataLen := samplePayload.DataLength()
			if dataLen == 0 {
				continue
			}

			// Extract frame data
			frameData := make([]byte, dataLen)
			for i := 0; i < int(dataLen); i++ {
				frameData[i] = byte(samplePayload.Data(i))
			}

			// Propagate the parent-provided media timestamp for better A/V sync.
			rtcConnection.PushAudioPcmData(
				frameData,
				int(initSampleRate),
				int(initAudioChannels),
				samplePayload.TimestampUnixNano()/1_000_000,
			)

		case ipcgen.MessageTypeCLOSE_COMMAND:
			childLogger.Println("Received Close command. Cleaning up and exiting.")
			cleanupAgoraResources()
			sendAsyncLogResponse(ipcgen.LogLevelINFO, "Child process shutting down.")
			sendAsyncStatusResponse(ipcgen.ConnectionStatusDISCONNECTED, "", "Closed by parent command")
			childLogger.Println("Child process terminated by close command.")
			return

		default:
			errMsg := fmt.Sprintf("Unknown command type received: %s", ipcgen.EnumNamesMessageType[ipcMsg.MessageType()])
			childLogger.Println(errMsg)
			sendErrorResponse(ipcgen.ConnectionStatusFAILED, errMsg, "")
		}
	}
}

func setupMediaInfrastructureAndPublish(conn *agoraservice.RtcConnection) error {
	if conn == nil {
		return fmt.Errorf("RtcConnection is nil in setupMediaInfrastructureAndPublish")
	}

	videoEncoderConfig := &agoraservice.VideoEncoderConfiguration{
		CodecType:         initVideoCodec,
		Width:             int(initWidth),
		Height:            int(initHeight),
		Framerate:         int(initFrameRate),
		Bitrate:           initBitrate,
		MinBitrate:        initMinBitrate,
		OrientationMode:   agoraservice.OrientationModeAdaptive,
		DegradePreference: agoraservice.DegradeMaintainBalanced,
	}

	// Apply codec-specific optimizations
	if initVideoCodec == agoraservice.VideoCodecTypeAv1 {
		childLogger.Printf("Applying AV1-specific optimizations: bitrate=%d, minBitrate=%d",
			videoEncoderConfig.Bitrate, videoEncoderConfig.MinBitrate)
		// AV1 can be more CPU intensive, so we might want to limit resolution for performance
		if initWidth > 1280 || initHeight > 720 {
			childLogger.Println("INFO: For optimal AV1 performance, consider using 720p or lower resolution")
		}
	}

	childLogger.Printf("Setting video encoder configuration: Codec=%s (enum=%d), %dx%d@%dfps, Bitrate=%d-%d Kbps",
		globalCodecName, videoEncoderConfig.CodecType, videoEncoderConfig.Width, videoEncoderConfig.Height,
		videoEncoderConfig.Framerate, videoEncoderConfig.MinBitrate, videoEncoderConfig.Bitrate)

	ret := conn.SetVideoEncoderConfiguration(videoEncoderConfig)
	if ret != 0 {
		errMsg := fmt.Sprintf("failed to set video encoder configuration for %s codec (enum=%d), error code: %d",
			globalCodecName, initVideoCodec, ret)
		childLogger.Printf("ERROR: %s", errMsg)
		return fmt.Errorf(errMsg)
	}
	childLogger.Printf("Video encoder configuration set successfully for %s codec (enum=%d).", globalCodecName, initVideoCodec)

	// Publish Audio and Video
	childLogger.Println("Publishing audio...")
	if ret := conn.PublishAudio(); ret != 0 {
		errMsg := fmt.Sprintf("failed to publish audio, error code: %d", ret)
		return fmt.Errorf(errMsg)
	}
	childLogger.Println("Audio published.")

	childLogger.Println("Publishing video...")
	if ret := conn.PublishVideo(); ret != 0 {
		errMsg := fmt.Sprintf("failed to publish video, error code: %d", ret)
		conn.UnpublishAudio()
		return fmt.Errorf(errMsg)
	}
	childLogger.Printf("Video published with %s codec (enum=%d).", globalCodecName, initVideoCodec)

	childLogger.Printf("Media infrastructure setup completed successfully. Streaming with %s codec (enum=%d) at %dx%d@%dfps, %d-%d Kbps",
		globalCodecName, initVideoCodec, initWidth, initHeight, initFrameRate, initMinBitrate, initBitrate)
	return nil
}

func cleanupAgoraResources() {
	childLogger.Println("Cleaning up ALL Agora resources due to CLOSE command or fatal error...")
	cleanupLocalRtcResources(true)
	childLogger.Println("Full Agora resources cleanup attempt finished.")
}
