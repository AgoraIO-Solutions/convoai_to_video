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

	"go-publish-video/ipc/ipcgen"

	agoraservice "github.com/AgoraIO-Extensions/Agora-Golang-Server-SDK/v2/go_sdk/agoraservice"
)

var (
	mediaFactory    *agoraservice.MediaNodeFactory
	videoSender     *agoraservice.VideoFrameSender
	audioSender     *agoraservice.AudioPcmDataSender
	localVideoTrack *agoraservice.LocalVideoTrack
	localAudioTrack *agoraservice.LocalAudioTrack
	rtcConnection   *agoraservice.RtcConnection

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

	videoFramesSent int64
	audioFramesSent int64
)

func onConnected(conn *agoraservice.RtcConnection, conInfo *agoraservice.RtcConnectionInfo, reason int) {
	logMsg := fmt.Sprintf("Agora SDK: Connected. UserID: %s, Channel: %s, Reason: %d", conInfo.LocalUserId, conInfo.ChannelId, reason)
	childLogger.Println(logMsg)
	sendAsyncLogResponse(ipcgen.LogLevelINFO, logMsg)

	if err := setupMediaInfrastructureAndPublish(conn); err != nil {
		errMsg := fmt.Sprintf("Failed to setup media infrastructure: %v", err)
		childLogger.Println("ERROR: " + errMsg)
		sendAsyncErrorResponse(ipcgen.ConnectionStatusFAILED, errMsg, "MediaSetupError")
		return
	}

	sendAsyncStatusResponse(ipcgen.ConnectionStatusCONNECTED, "Successfully connected and media infrastructure prepared.", "")
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

	childLogger.Printf("Initial parameters: AppID=%s, Channel=%s, UserID=%s, Codec=%s, Res=%dx%d@%d, Bitrate=%dKbps, MinBitrate=%dKbps, AudioSR=%d, AudioCh=%d, StringUID=%t",
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

	switch *videoCodecFlag {
	case "H264":
		initVideoCodec = agoraservice.VideoCodecTypeH264
	case "VP8":
		initVideoCodec = agoraservice.VideoCodecTypeVp8
	default:
		childLogger.Printf("WARN: Unsupported video_codec_name %q, defaulting to H264", *videoCodecFlag)
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
		sendErrorResponse(ipcgen.ConnectionStatusINITIALIZED_FAILURE, errMsg, "RtcConnectionCreateFailed")
		os.Exit(1)
	}

	observer := &agoraservice.RtcConnectionObserver{
		OnConnected:                onConnected,
		OnDisconnected:             onDisconnected,
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

	if ret := rtcConnection.Connect(childProcessToken, globalChannel, globalUserID); ret != 0 {
		errMsg := fmt.Sprintf("Agora SDK Connect() failed with code: %d", ret)
		childLogger.Println("ERROR: " + errMsg)
		sendErrorResponse(ipcgen.ConnectionStatusINITIALIZED_FAILURE, errMsg, "ConnectFailed")
		cleanupAgoraResources()
		os.Exit(1)
	}
	childLogger.Println("Connect() called successfully; waiting for OnConnected callback.")
	sendStatusResponse(ipcgen.ConnectionStatusINITIALIZED_SUCCESS, "Agora SDK initialized and connect initiated.", "")

	reader := bufio.NewReader(os.Stdin)
	for {
		lenBytes := make([]byte, 4)
		if _, err := io.ReadFull(reader, lenBytes); err != nil {
			if err == io.EOF {
				childLogger.Println("Parent stdin closed. Cleaning up and exiting.")
				cleanupAgoraResources()
				return
			}
			errMsg := fmt.Sprintf("Failed to read message length from parent: %v", err)
			childLogger.Println("ERROR: " + errMsg)
			sendAsyncErrorResponse(ipcgen.ConnectionStatusFAILED, errMsg, "ParentCommReadLengthError")
			cleanupAgoraResources()
			return
		}

		msgLen := binary.BigEndian.Uint32(lenBytes)
		if msgLen == 0 {
			continue
		}

		msgBuf := make([]byte, msgLen)
		if _, err := io.ReadFull(reader, msgBuf); err != nil {
			errMsg := fmt.Sprintf("Failed to read message payload from parent: %v", err)
			childLogger.Println("ERROR: " + errMsg)
			sendAsyncErrorResponse(ipcgen.ConnectionStatusFAILED, errMsg, "ParentCommReadPayloadError")
			cleanupAgoraResources()
			return
		}

		ipcMsg := ipcgen.GetRootAsIPCMessage(msgBuf, 0)
		payloadLen := ipcMsg.PayloadLength()
		if payloadLen == 0 && ipcMsg.MessageType() != ipcgen.MessageTypeCLOSE_COMMAND {
			childLogger.Printf("No payload for message type: %s", ipcgen.EnumNamesMessageType[ipcMsg.MessageType()])
			continue
		}

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

			samplePayload := ipcgen.GetRootAsMediaSamplePayload(payloadBytes, 0)
			dataLen := samplePayload.DataLength()
			if dataLen == 0 {
				childLogger.Println("WARN: Received empty video sample data.")
				continue
			}

			frameData := make([]byte, dataLen)
			for i := 0; i < int(dataLen); i++ {
				frameData[i] = byte(samplePayload.Data(i))
			}

			timestampMs := samplePayload.TimestampUnixNano() / 1_000_000
			extFrame := &agoraservice.ExternalVideoFrame{
				Type:      agoraservice.VideoBufferRawData,
				Format:    agoraservice.VideoPixelRGBA,
				Buffer:    frameData,
				Stride:    int(initWidth),
				Height:    int(initHeight),
				Timestamp: timestampMs,
			}
			ret := videoSender.SendVideoFrame(extFrame)
			videoFramesSent++
			if videoFramesSent <= 5 || videoFramesSent%30 == 0 {
				childLogger.Printf("SendVideoFrame frame=%d bytes=%d timestamp_ms=%d ret=%d", videoFramesSent, len(frameData), timestampMs, ret)
			}
			if ret != 0 {
				childLogger.Printf("WARN: videoSender.SendVideoFrame failed, error code: %d", ret)
			}

		case ipcgen.MessageTypeWRITE_AUDIO_SAMPLE_COMMAND:
			if rtcConnection == nil || audioSender == nil {
				childLogger.Println("WARN: Audio sample received but Agora rtcConnection/audio sender not ready. Dropping.")
				continue
			}

			samplePayload := ipcgen.GetRootAsMediaSamplePayload(payloadBytes, 0)
			dataLen := samplePayload.DataLength()
			if dataLen == 0 {
				childLogger.Println("WARN: Received empty audio sample data.")
				continue
			}

			frameData := make([]byte, dataLen)
			for i := 0; i < int(dataLen); i++ {
				frameData[i] = byte(samplePayload.Data(i))
			}

			bytesPerSample := 2
			if initAudioChannels == 0 {
				childLogger.Println("ERROR: initAudioChannels is 0, cannot calculate audio frame shape.")
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
				RenderTimeMs:      samplePayload.TimestampUnixNano() / 1_000_000,
			}
			ret := audioSender.SendAudioPcmData(audioFrame)
			audioFramesSent++
			if audioFramesSent <= 5 || audioFramesSent%100 == 0 {
				childLogger.Printf("SendAudioPcmData frame=%d bytes=%d samples_per_channel=%d render_time_ms=%d ret=%d", audioFramesSent, len(frameData), samplesPerChannel, audioFrame.RenderTimeMs, ret)
			}
			if ret != 0 {
				childLogger.Printf("WARN: audioSender.SendAudioPcmData failed, error code: %d", ret)
			}

		case ipcgen.MessageTypeCLOSE_COMMAND:
			childLogger.Println("Received Close command. Cleaning up and exiting.")
			cleanupAgoraResources()
			sendAsyncLogResponse(ipcgen.LogLevelINFO, "Child process shutting down.")
			sendAsyncStatusResponse(ipcgen.ConnectionStatusDISCONNECTED, "", "Closed by parent command")
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

	childLogger.Println("Creating AudioPcmDataSender...")
	audioSender = mediaFactory.NewAudioPcmDataSender()
	if audioSender == nil {
		return fmt.Errorf("failed to create AudioPcmDataSender")
	}

	childLogger.Println("Creating VideoFrameSender...")
	videoSender = mediaFactory.NewVideoFrameSender()
	if videoSender == nil {
		audioSender.Release()
		audioSender = nil
		return fmt.Errorf("failed to create VideoFrameSender")
	}

	childLogger.Println("Creating custom audio track (PCM)...")
	localAudioTrack = agoraservice.NewCustomAudioTrackPcm(audioSender)
	if localAudioTrack == nil {
		audioSender.Release()
		audioSender = nil
		videoSender.Release()
		videoSender = nil
		return fmt.Errorf("failed to create custom audio track (PCM)")
	}

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
		cleanupLocalRtcResources(false)
		return fmt.Errorf("failed to set video encoder configuration, error code: %d", ret)
	}

	localAudioTrack.SetEnabled(true)
	localVideoTrack.SetEnabled(true)

	childLogger.Println("Publishing local audio track...")
	if ret := localUser.PublishAudio(localAudioTrack); ret != 0 {
		cleanupLocalRtcResources(false)
		return fmt.Errorf("failed to publish audio track, error code: %d", ret)
	}

	childLogger.Println("Publishing local video track...")
	if ret := localUser.PublishVideo(localVideoTrack); ret != 0 {
		localUser.UnpublishAudio(localAudioTrack)
		cleanupLocalRtcResources(false)
		return fmt.Errorf("failed to publish video track, error code: %d", ret)
	}

	childLogger.Println("Media infrastructure setup and publishing completed successfully.")
	return nil
}

func cleanupAgoraResources() {
	childLogger.Println("Cleaning up ALL Agora resources due to CLOSE command or fatal error...")
	cleanupLocalRtcResources(true)
	childLogger.Println("Full Agora resources cleanup attempt finished.")
}
