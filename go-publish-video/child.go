package main

import (
	"bufio"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"sync"

	"agora-go-publish/ipc/ipcgen"

	agoraservice "github.com/AgoraIO-Extensions/Agora-Golang-Server-SDK/v2/go_sdk/agoraservice"
	flatbuffers "github.com/google/flatbuffers/go"
)

var (
	childLogger  *log.Logger
	stdoutWriter *bufio.Writer
	stdoutLock   sync.Mutex

	// Global Agora SDK objects
	mediaFactory    *agoraservice.MediaNodeFactory
	videoSender     *agoraservice.VideoFrameSender
	audioSender     *agoraservice.AudioPcmDataSender
	localVideoTrack *agoraservice.LocalVideoTrack
	localAudioTrack *agoraservice.LocalAudioTrack

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
		sendAsyncStatusResponse(ipcgen.ConnectionStatusCONNECTED, "Successfully connected and media infrastructure prepared.", "")
	}
}

func onDisconnected(conn *agoraservice.RtcConnection, conInfo *agoraservice.RtcConnectionInfo, reason int) {
	logMsg := fmt.Sprintf("Agora SDK: Disconnected. Reason: %d", reason)
	childLogger.Println(logMsg)
	sendAsyncLogResponse(ipcgen.LogLevelWARN, logMsg)
	sendAsyncStatusResponse(ipcgen.ConnectionStatusDISCONNECTED, logMsg, "")
	cleanupLocalRtcResources(false)
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
	cleanupLocalRtcResources(false)
}

func onConnectionFailure(conn *agoraservice.RtcConnection, conInfo *agoraservice.RtcConnectionInfo, errCode int) {
	logMsg := fmt.Sprintf("Agora SDK: Connection failure. Error Code: %d", errCode)
	childLogger.Println("ERROR: " + logMsg)
	sendAsyncLogResponse(ipcgen.LogLevelERROR, logMsg)
	sendAsyncErrorResponse(ipcgen.ConnectionStatusFAILED, logMsg, fmt.Sprintf("AgoraErrorCode: %d", errCode))
	cleanupLocalRtcResources(false)
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
		localUser := rtcConnection.GetLocalUser()
		if localUser != nil {
			if localVideoTrack != nil {
				childLogger.Println("Unpublishing video track...")
				localUser.UnpublishVideo(localVideoTrack)
				childLogger.Println("Releasing local video track...")
				localVideoTrack.Release()
				localVideoTrack = nil
			}
			if localAudioTrack != nil {
				childLogger.Println("Unpublishing audio track...")
				localUser.UnpublishAudio(localAudioTrack)
				childLogger.Println("Releasing local audio track...")
				localAudioTrack.Release()
				localAudioTrack = nil
			}
		}
	}

	if videoSender != nil {
		childLogger.Println("Releasing video sender...")
		videoSender.Release()
		videoSender = nil
	}
	if audioSender != nil {
		childLogger.Println("Releasing audio sender...")
		audioSender.Release()
		audioSender = nil
	}

	if rtcConnection != nil {
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
	childLogger = log.New(os.Stderr, "[agora_worker] ", log.LstdFlags|log.Lshortfile)
	childLogger.Println("Agora child process started.")
	stdoutWriter = bufio.NewWriter(os.Stdout)

	// Define command-line flags
	appIDFlag := flag.String("appID", "", "Agora App ID")
	channelNameFlag := flag.String("channelName", "", "Agora Channel Name")
	userIDFlag := flag.String("userID", "", "Agora User ID for the child process")
	tokenFlag := flag.String("token", "", "Agora Token for the child process")
	widthFlag := flag.Int("width", 352, "Video width")
	heightFlag := flag.Int("height", 288, "Video height")
	frameRateFlag := flag.Int("frameRate", 15, "Video frame rate")
	videoCodecFlag := flag.String("videoCodec", "H264", "Video codec (H264 or VP8)")
	sampleRateFlag := flag.Int("sampleRate", 16000, "Audio sample rate")
	audioChannelsFlag := flag.Int("audioChannels", 1, "Audio channels")
	bitrateFlag := flag.Int("bitrate", 1000, "Video target bitrate in Kbps")
	minBitrateFlag := flag.Int("minBitrate", 100, "Video minimum bitrate in Kbps")
	enableStringUIDFlag := flag.Bool("enableStringUID", false, "Enable string UID support")

	flag.Parse()

	globalAppID = *appIDFlag
	globalChannel = *channelNameFlag
	globalUserID = *userIDFlag
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

	serviceCfg := agoraservice.NewAgoraServiceConfig()
	serviceCfg.EnableAudioProcessor = true
	serviceCfg.EnableVideo = true
	serviceCfg.AppId = globalAppID
	serviceCfg.UseStringUid = enableStringUID
	serviceCfg.LogPath = "./agora_child_sdk.log"
	serviceCfg.LogSize = 5 * 1024 * 1024

	if ret := agoraservice.Initialize(serviceCfg); ret != 0 {
		errMsg := fmt.Sprintf("Agora SDK global Initialize() failed with code: %d", ret)
		childLogger.Println("FATAL: " + errMsg)
		sendErrorResponse(ipcgen.ConnectionStatusINITIALIZED_FAILURE, errMsg, "GlobalInitializeFailed")
		os.Exit(1)
	}
	childLogger.Println("Agora SDK global Initialize() successful.")
	defer agoraservice.Release()

	mediaFactory = agoraservice.NewMediaNodeFactory()
	if mediaFactory == nil {
		childLogger.Println("FATAL: Failed to create MediaNodeFactory")
		sendErrorResponse(ipcgen.ConnectionStatusINITIALIZED_FAILURE, "Failed to create MediaNodeFactory", "")
		os.Exit(1)
	}
	childLogger.Println("MediaNodeFactory created.")

	// Determine video codec type from flags
	switch *videoCodecFlag {
	case "H264":
		initVideoCodec = agoraservice.VideoCodecTypeH264
	case "VP8":
		initVideoCodec = agoraservice.VideoCodecTypeVp8
	default:
		childLogger.Printf("WARN: Unsupported video_codec_name '%s' from CLI, defaulting to H264 for Agora.", *videoCodecFlag)
		initVideoCodec = agoraservice.VideoCodecTypeH264
	}

	connCfg := &agoraservice.RtcConnectionConfig{
		AutoSubscribeAudio: false,
		AutoSubscribeVideo: false,
		ClientRole:         agoraservice.ClientRoleBroadcaster,
		ChannelProfile:     agoraservice.ChannelProfileLiveBroadcasting,
	}

	rtcConnection = agoraservice.NewRtcConnection(connCfg)
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
	
	if ret := rtcConnection.RegisterObserver(observer); ret != 0 {
		errMsg := fmt.Sprintf("Failed to register RtcConnectionObserver, error code: %d", ret)
		childLogger.Println("ERROR: " + errMsg)
		sendErrorResponse(ipcgen.ConnectionStatusINITIALIZED_FAILURE, errMsg, "RegisterObserverFailed")
		rtcConnection.Release()
		rtcConnection = nil
		os.Exit(1)
	}
	childLogger.Println("Agora RtcConnection created and observer registered.")

	ret := rtcConnection.Connect(childProcessToken, globalChannel, globalUserID)
	if ret != 0 {
		errMsg := fmt.Sprintf("Agora RtcConnection.Connect() call failed with code: %d", ret)
		childLogger.Println("ERROR: " + errMsg)
		rtcConnection.UnregisterObserver()
		rtcConnection.Release()
		rtcConnection = nil
		sendErrorResponse(ipcgen.ConnectionStatusINITIALIZED_FAILURE, errMsg, "ConnectFailed")
		os.Exit(1)
	}
	childLogger.Printf("Agora RtcConnection.Connect() called for channel '%s', user '%s'. Waiting for connection callbacks.", globalChannel, globalUserID)
	sendStatusResponse(ipcgen.ConnectionStatusINITIALIZED_SUCCESS, "Connect call issued, awaiting callback.", "")

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
			if rtcConnection == nil || videoSender == nil {
				childLogger.Println("WARN: Video sample received but Agora rtcConnection/video sender not ready. Dropping.")
				continue
			}
			
			// Parse MediaSamplePayload from payload bytes
			samplePayload := ipcgen.GetRootAsMediaSamplePayload(payloadBytes, 0)
			dataLen := samplePayload.DataLength()
			if dataLen == 0 {
				childLogger.Println("WARN: Received empty video sample data.")
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
				Timestamp: int64(0),
			}
			if ret := videoSender.SendVideoFrame(extFrame); ret != 0 {
				childLogger.Printf("WARN: videoSender.SendVideoFrame failed, error code: %d", ret)
			}

		case ipcgen.MessageTypeWRITE_AUDIO_SAMPLE_COMMAND:
			if rtcConnection == nil || audioSender == nil {
				childLogger.Println("WARN: Audio sample received but Agora rtcConnection/audio sender not ready. Dropping.")
				continue
			}
			
			// Parse MediaSamplePayload from payload bytes
			samplePayload := ipcgen.GetRootAsMediaSamplePayload(payloadBytes, 0)
			dataLen := samplePayload.DataLength()
			if dataLen == 0 {
				childLogger.Println("WARN: Received empty audio sample data.")
				continue
			}
			
			// Extract frame data
			frameData := make([]byte, dataLen)
			for i := 0; i < int(dataLen); i++ {
				frameData[i] = byte(samplePayload.Data(i))
			}

			bytesPerSample := 2 // For PCM16
			if initAudioChannels == 0 {
				childLogger.Println("ERROR: initAudioChannels is 0, cannot calculate samplesPerChannel for audio frame.")
				continue
			}
			samplesPerChannel := len(frameData) / (int(initAudioChannels) * bytesPerSample)

			audioFrame := &agoraservice.AudioFrame{
				Type:              agoraservice.AudioFrameTypePCM16,
				SamplesPerChannel: samplesPerChannel,
				BytesPerSample:    bytesPerSample,
				Channels:          int(initAudioChannels),
				SamplesPerSec:     int(initSampleRate),
				Buffer:            frameData,
				RenderTimeMs:      int64(0),
			}
			if ret := audioSender.SendAudioPcmData(audioFrame); ret != 0 {
				childLogger.Printf("WARN: audioSender.SendAudioPcmData failed, error code: %d", ret)
			}

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
	localUser := conn.GetLocalUser()
	if localUser == nil {
		return fmt.Errorf("LocalUser is nil in setupMediaInfrastructureAndPublish")
	}
	if mediaFactory == nil {
		return fmt.Errorf("MediaNodeFactory is nil in setupMediaInfrastructureAndPublish")
	}

	// Create Senders
	childLogger.Println("Creating AudioPcmDataSender...")
	audioSender = mediaFactory.NewAudioPcmDataSender()
	if audioSender == nil {
		return fmt.Errorf("failed to create AudioPcmDataSender")
	}
	childLogger.Println("AudioPcmDataSender created.")

	childLogger.Println("Creating VideoFrameSender...")
	videoSender = mediaFactory.NewVideoFrameSender()
	if videoSender == nil {
		audioSender.Release()
		audioSender = nil
		return fmt.Errorf("failed to create VideoFrameSender")
	}
	childLogger.Println("VideoFrameSender created.")

	// Create Tracks
	childLogger.Println("Creating custom audio track (PCM)...")
	localAudioTrack = agoraservice.NewCustomAudioTrackPcm(audioSender)
	if localAudioTrack == nil {
		audioSender.Release()
		audioSender = nil
		videoSender.Release()
		videoSender = nil
		return fmt.Errorf("failed to create custom audio track (PCM)")
	}
	childLogger.Println("Custom audio track (PCM) created.")

	childLogger.Println("Creating custom video track (Frame)...")
	localVideoTrack = agoraservice.NewCustomVideoTrackFrame(videoSender)
	if localVideoTrack == nil {
		localAudioTrack.Release()
		localAudioTrack = nil
		audioSender.Release()
		audioSender = nil
		videoSender.Release()
		videoSender = nil
		return fmt.Errorf("failed to create custom video track (Frame)")
	}
	childLogger.Println("Custom video track (Frame) created.")

	// Configure Video Track
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
	childLogger.Printf("Setting video encoder configuration: %+v", videoEncoderConfig)
	if ret := localVideoTrack.SetVideoEncoderConfiguration(videoEncoderConfig); ret != 0 {
		errMsg := fmt.Sprintf("failed to set video encoder configuration, error code: %d", ret)
		cleanupLocalRtcResources(false)
		return fmt.Errorf(errMsg)
	}
	childLogger.Println("Video encoder configuration set.")

	// Enable Tracks
	childLogger.Println("Enabling local audio track...")
	localAudioTrack.SetEnabled(true)
	childLogger.Println("Enabling local video track...")
	localVideoTrack.SetEnabled(true)

	// Publish Tracks
	childLogger.Println("Publishing local audio track...")
	if ret := localUser.PublishAudio(localAudioTrack); ret != 0 {
		errMsg := fmt.Sprintf("failed to publish audio track, error code: %d", ret)
		cleanupLocalRtcResources(false)
		return fmt.Errorf(errMsg)
	}
	childLogger.Println("Local audio track published.")

	childLogger.Println("Publishing local video track...")
	if ret := localUser.PublishVideo(localVideoTrack); ret != 0 {
		errMsg := fmt.Sprintf("failed to publish video track, error code: %d", ret)
		localUser.UnpublishAudio(localAudioTrack)
		cleanupLocalRtcResources(false)
		return fmt.Errorf(errMsg)
	}
	childLogger.Println("Local video track published.")

	childLogger.Println("Media infrastructure setup and publishing completed successfully.")
	return nil
}

func cleanupAgoraResources() {
	childLogger.Println("Cleaning up ALL Agora resources due to CLOSE command or fatal error...")
	cleanupLocalRtcResources(true)
	childLogger.Println("Full Agora resources cleanup attempt finished.")
}

func sendAsyncStatusResponse(status ipcgen.ConnectionStatus, message string, details string) {
	stdoutLock.Lock()
	defer stdoutLock.Unlock()

	// First create the StatusResponsePayload
	innerBuilder := flatbuffers.NewBuilder(1024)
	msgStr := innerBuilder.CreateString(message)
	detailsStr := innerBuilder.CreateString(details)

	ipcgen.StatusResponsePayloadStart(innerBuilder)
	ipcgen.StatusResponsePayloadAddStatus(innerBuilder, status)
	ipcgen.StatusResponsePayloadAddErrorMessage(innerBuilder, msgStr)
	ipcgen.StatusResponsePayloadAddAdditionalInfo(innerBuilder, detailsStr)
	statusPayloadOffset := ipcgen.StatusResponsePayloadEnd(innerBuilder)
	innerBuilder.Finish(statusPayloadOffset)
	
	// Get the serialized StatusResponsePayload bytes
	statusPayloadBytes := innerBuilder.FinishedBytes()
	
	// Now create the outer IPCMessage with the StatusResponsePayload bytes as payload
	outerBuilder := flatbuffers.NewBuilder(len(statusPayloadBytes) + 64)
	
	// Create payload vector for IPCMessage
	ipcgen.IPCMessageStartPayloadVector(outerBuilder, len(statusPayloadBytes))
	for i := len(statusPayloadBytes) - 1; i >= 0; i-- {
		outerBuilder.PrependByte(statusPayloadBytes[i])
	}
	payloadOffset := outerBuilder.EndVector(len(statusPayloadBytes))
	
	// Create IPCMessage
	ipcgen.IPCMessageStart(outerBuilder)
	ipcgen.IPCMessageAddMessageType(outerBuilder, ipcgen.MessageTypeSTATUS_RESPONSE)
	ipcgen.IPCMessageAddPayloadType(outerBuilder, ipcgen.MessagePayloadStatus)
	ipcgen.IPCMessageAddPayload(outerBuilder, payloadOffset)
	msg := ipcgen.IPCMessageEnd(outerBuilder)
	outerBuilder.Finish(msg)

	buf := outerBuilder.FinishedBytes()
	sendFramedMessage(stdoutWriter, buf)
	if err := stdoutWriter.Flush(); err != nil {
		childLogger.Printf("ERROR flushing stdout after status response: %v", err)
	}
}

func sendAsyncErrorResponse(statusForError ipcgen.ConnectionStatus, errMsgStr string, errorDetails string) {
	sendAsyncStatusResponse(statusForError, errMsgStr, errorDetails)
}

func sendAsyncLogResponse(level ipcgen.LogLevel, messageStr string) {
	stdoutLock.Lock()
	defer stdoutLock.Unlock()

	// First create the LogResponsePayload
	innerBuilder := flatbuffers.NewBuilder(1024)
	msgStr := innerBuilder.CreateString(messageStr)

	ipcgen.LogResponsePayloadStart(innerBuilder)
	ipcgen.LogResponsePayloadAddLevel(innerBuilder, level)
	ipcgen.LogResponsePayloadAddMessage(innerBuilder, msgStr)
	logPayloadOffset := ipcgen.LogResponsePayloadEnd(innerBuilder)
	innerBuilder.Finish(logPayloadOffset)
	
	// Get the serialized LogResponsePayload bytes
	logPayloadBytes := innerBuilder.FinishedBytes()
	
	// Now create the outer IPCMessage with the LogResponsePayload bytes as payload
	outerBuilder := flatbuffers.NewBuilder(len(logPayloadBytes) + 64)
	
	// Create payload vector for IPCMessage
	ipcgen.IPCMessageStartPayloadVector(outerBuilder, len(logPayloadBytes))
	for i := len(logPayloadBytes) - 1; i >= 0; i-- {
		outerBuilder.PrependByte(logPayloadBytes[i])
	}
	payloadOffset := outerBuilder.EndVector(len(logPayloadBytes))
	
	// Create IPCMessage
	ipcgen.IPCMessageStart(outerBuilder)
	ipcgen.IPCMessageAddMessageType(outerBuilder, ipcgen.MessageTypeLOG_RESPONSE)
	ipcgen.IPCMessageAddPayloadType(outerBuilder, ipcgen.MessagePayloadLog)
	ipcgen.IPCMessageAddPayload(outerBuilder, payloadOffset)
	msg := ipcgen.IPCMessageEnd(outerBuilder)
	outerBuilder.Finish(msg)

	buf := outerBuilder.FinishedBytes()
	sendFramedMessage(stdoutWriter, buf)
	if err := stdoutWriter.Flush(); err != nil {
		childLogger.Printf("ERROR flushing stdout after log response: %v", err)
	}
}

func sendStatusResponse(status ipcgen.ConnectionStatus, errMsgStr string, addInfoStr string) {
	sendAsyncStatusResponse(status, errMsgStr, addInfoStr)
}

func sendErrorResponse(statusForError ipcgen.ConnectionStatus, errorMessage string, errorDetails string) {
	sendAsyncStatusResponse(statusForError, errorMessage, errorDetails)
}

func sendFramedMessage(writer *bufio.Writer, msg []byte) {
	lenBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(lenBytes, uint32(len(msg)))

	if _, err := writer.Write(lenBytes); err != nil {
		childLogger.Printf("Failed to write message length to writer: %v", err)
		return
	}
	if _, err := writer.Write(msg); err != nil {
		childLogger.Printf("Failed to write message payload to writer: %v", err)
	}
}
