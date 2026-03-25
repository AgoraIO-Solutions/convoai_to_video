//go:build !test

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"sync"
	"sync/atomic"
	"time"

	agoraservice "github.com/AgoraIO-Extensions/Agora-Golang-Server-SDK/v2/go_sdk/rtc"
	rtctokenbuilder "github.com/AgoraIO/Tools/DynamicKey/AgoraDynamicKey/go/src/rtctokenbuilder2"
)

func main() {
	var (
		appID          string
		appCert        string
		channelName    string
		publisherUID   string
		enableStringUID bool
	)

	flag.StringVar(&appID, "appID", "", "Agora App ID (required)")
	flag.StringVar(&appCert, "appCert", "", "App certificate for token generation (required)")
	flag.StringVar(&channelName, "channelName", "", "Channel name to join (required)")
	flag.StringVar(&publisherUID, "publisherUID", "", "Publisher UID to expect audio from")
	flag.BoolVar(&enableStringUID, "enableStringUID", true, "Enable string UID support")
	flag.Parse()

	if appID == "" || appCert == "" || channelName == "" {
		fmt.Println("Error: -appID, -appCert, and -channelName are required")
		flag.Usage()
		os.Exit(1)
	}

	logger := log.New(os.Stderr, "[subscriber] ", log.LstdFlags|log.Lshortfile)

	// Determine subscriber UID
	subscriberUID := "subscriber_200"
	if !enableStringUID {
		subscriberUID = "200"
	}

	// Generate subscriber token
	token, err := rtctokenbuilder.BuildTokenWithUserAccount(
		appID, appCert, channelName, subscriberUID,
		rtctokenbuilder.RoleSubscriber, 3600, 3600,
	)
	if err != nil {
		logger.Fatalf("Failed to generate subscriber token: %v", err)
	}
	fmt.Printf("[subscriber] Generated subscriber token for uid=%s\n", subscriberUID)

	// Initialize Agora service
	serviceCfg := agoraservice.NewAgoraServiceConfig()
	serviceCfg.EnableAudioProcessor = true
	serviceCfg.EnableVideo = true
	serviceCfg.AppId = appID
	serviceCfg.UseStringUid = enableStringUID
	serviceCfg.LogPath = "./agora_subscriber_sdk.log"
	serviceCfg.LogSize = 5 * 1024 * 1024
	serviceCfg.LogLevel = 5 // Error only

	if ret := agoraservice.Initialize(serviceCfg); ret != 0 {
		logger.Fatalf("Agora SDK Initialize() failed with code: %d", ret)
	}
	defer agoraservice.Release()
	logger.Println("Agora SDK initialized.")

	// Connection configuration — subscribe to audio only, don't publish
	connCfg := &agoraservice.RtcConnectionConfig{
		AutoSubscribeAudio: true,
		AutoSubscribeVideo: false,
		ClientRole:         agoraservice.ClientRoleBroadcaster,
		ChannelProfile:     agoraservice.ChannelProfileLiveBroadcasting,
	}

	publishConfig := agoraservice.NewRtcConPublishConfig()
	publishConfig.IsPublishAudio = false
	publishConfig.IsPublishVideo = false

	conn := agoraservice.NewRtcConnection(connCfg, publishConfig)
	if conn == nil {
		logger.Fatalf("Failed to create RtcConnection")
	}

	// Channels for synchronization
	connectedChan := make(chan struct{}, 1)
	var frameCount int64

	// Connection observer
	var connectedOnce sync.Once
	connObserver := &agoraservice.RtcConnectionObserver{
		OnConnected: func(c *agoraservice.RtcConnection, info *agoraservice.RtcConnectionInfo, reason int) {
			logger.Printf("Connected to channel %s as %s", info.ChannelId, info.LocalUserId)
			fmt.Printf("[subscriber] Connected to channel %s\n", info.ChannelId)
			connectedOnce.Do(func() { close(connectedChan) })
		},
		OnDisconnected: func(c *agoraservice.RtcConnection, info *agoraservice.RtcConnectionInfo, reason int) {
			logger.Printf("Disconnected. Reason: %d", reason)
		},
		OnConnecting: func(c *agoraservice.RtcConnection, info *agoraservice.RtcConnectionInfo, reason int) {
			logger.Printf("Connecting...")
		},
		OnReconnecting: func(c *agoraservice.RtcConnection, info *agoraservice.RtcConnectionInfo, reason int) {
			logger.Printf("Reconnecting... Reason: %d", reason)
		},
		OnReconnected: func(c *agoraservice.RtcConnection, info *agoraservice.RtcConnectionInfo, reason int) {
			logger.Printf("Reconnected.")
		},
		OnConnectionLost: func(c *agoraservice.RtcConnection, info *agoraservice.RtcConnectionInfo) {
			logger.Printf("Connection lost.")
		},
		OnConnectionFailure: func(c *agoraservice.RtcConnection, info *agoraservice.RtcConnectionInfo, errCode int) {
			logger.Printf("Connection failure. Error: %d", errCode)
		},
		OnTokenPrivilegeWillExpire: func(c *agoraservice.RtcConnection, t string) {
			logger.Printf("Token will expire soon.")
		},
		OnTokenPrivilegeDidExpire: func(c *agoraservice.RtcConnection) {
			logger.Printf("Token expired.")
		},
		OnUserJoined: func(c *agoraservice.RtcConnection, uid string) {
			logger.Printf("User joined: %s", uid)
			fmt.Printf("[subscriber] User joined: %s\n", uid)
		},
		OnUserLeft: func(c *agoraservice.RtcConnection, uid string, reason int) {
			logger.Printf("User left: %s, reason: %d", uid, reason)
		},
		OnError: func(c *agoraservice.RtcConnection, errCode int, msg string) {
			logger.Printf("Error: %d, %s", errCode, msg)
		},
	}
	conn.RegisterObserver(connObserver)

	// Local user observer
	localUserObserver := &agoraservice.LocalUserObserver{
		OnUserAudioTrackSubscribed: func(localUser *agoraservice.LocalUser, uid string, remoteAudioTrack *agoraservice.RemoteAudioTrack) {
			logger.Printf("Audio track subscribed for uid: %s", uid)
			fmt.Printf("[subscriber] Audio track subscribed for uid: %s\n", uid)
		},
	}
	conn.RegisterLocalUserObserver(localUserObserver)

	// Audio frame observer
	audioObserver := &agoraservice.AudioFrameObserver{
		OnPlaybackAudioFrameBeforeMixing: func(localUser *agoraservice.LocalUser, channelId string, uid string, frame *agoraservice.AudioFrame, vadResultState agoraservice.VadState, vadResultFrame *agoraservice.AudioFrame) bool {
			count := atomic.AddInt64(&frameCount, 1)
			if count <= 10 || count%50 == 0 {
				logger.Printf("Received audio frame %d from uid: %s (samples=%d, rate=%d, ch=%d)",
					count, uid, frame.SamplesPerChannel, frame.SamplesPerSec, frame.Channels)
				fmt.Printf("[subscriber] Received audio frame %d from uid: %s\n", count, uid)
			}
			return true
		},
	}
	conn.RegisterAudioFrameObserver(audioObserver, 0, nil)

	// Set audio frame parameters
	localUser := conn.GetLocalUser()
	localUser.SetPlaybackAudioFrameBeforeMixingParameters(1, 16000)

	// Connect
	logger.Printf("Connecting to channel %s as %s...", channelName, subscriberUID)
	ret := conn.Connect(token, channelName, subscriberUID)
	if ret != 0 {
		logger.Fatalf("Connect() failed with code: %d", ret)
	}

	// Wait for connection (timeout 15s)
	select {
	case <-connectedChan:
		logger.Println("Connection established.")
	case <-time.After(15 * time.Second):
		logger.Println("FAIL: Timeout waiting for connection")
		fmt.Println("[subscriber] FAIL: Timeout waiting for connection")
		conn.Disconnect()
		conn.Release()
		os.Exit(1)
	}

	// Wait for audio frames (timeout 30s)
	deadline := time.After(30 * time.Second)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-deadline:
			count := atomic.LoadInt64(&frameCount)
			logger.Printf("FAIL: Timeout waiting for audio frames. Got %d frames (need >= 10)", count)
			fmt.Printf("[subscriber] FAIL: Only received %d audio frames (need >= 10)\n", count)
			conn.Disconnect()
			conn.Release()
			os.Exit(1)
		case <-ticker.C:
			count := atomic.LoadInt64(&frameCount)
			if count >= 10 {
				logger.Printf("PASS: Received %d audio frames from publisher", count)
				fmt.Printf("[subscriber] PASS: Received %d audio frames from publisher\n", count)
				conn.Disconnect()
				conn.Release()
				os.Exit(0)
			}
		}
	}
}
