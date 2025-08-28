package main

import (
	"bufio"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"agora-go-publish/ipc/ipcgen"
	flatbuffers "github.com/google/flatbuffers/go"
)

type ParentController struct {
	cmd          *exec.Cmd
	stdin        io.WriteCloser
	stdout       io.ReadCloser
	stderr       io.ReadCloser
	logger       *log.Logger
	mu           sync.Mutex
	isConnected  bool
	shutdownChan chan struct{}
	wg           sync.WaitGroup

	// Media configuration
	audioFile      string
	videoFile      string
	sampleRate     int
	audioChannels  int
	videoWidth     int
	videoHeight    int
	frameRate      int
	videoBitrate   int
	minVideoBitrate int
}

func NewParentController(opts *Options) *ParentController {
	return &ParentController{
		logger:         log.New(os.Stderr, "[parent] ", log.LstdFlags|log.Lshortfile),
		shutdownChan:   make(chan struct{}),
		audioFile:      opts.AudioFile,
		videoFile:      opts.VideoFile,
		sampleRate:     opts.SampleRate,
		audioChannels:  opts.AudioChannels,
		videoWidth:     opts.VideoWidth,
		videoHeight:    opts.VideoHeight,
		frameRate:      opts.FrameRate,
		videoBitrate:   opts.VideoBitrate,
		minVideoBitrate: opts.MinVideoBitrate,
	}
}

type Options struct {
	AppID          string
	ChannelName    string
	UserID         string
	Token          string
	AudioFile      string
	VideoFile      string
	SampleRate     int
	AudioChannels  int
	VideoWidth     int
	VideoHeight    int
	FrameRate      int
	VideoCodec     string
	VideoBitrate   int
	MinVideoBitrate int
	EnableStringUID bool
}

func (p *ParentController) Start(opts *Options) error {
	p.logger.Println("Starting child process...")

	// Build command with all necessary flags
	args := []string{
		"-appID", opts.AppID,
		"-channelName", opts.ChannelName,
		"-userID", opts.UserID,
		"-token", opts.Token,
		"-width", fmt.Sprintf("%d", opts.VideoWidth),
		"-height", fmt.Sprintf("%d", opts.VideoHeight),
		"-frameRate", fmt.Sprintf("%d", opts.FrameRate),
		"-videoCodec", opts.VideoCodec,
		"-sampleRate", fmt.Sprintf("%d", opts.SampleRate),
		"-audioChannels", fmt.Sprintf("%d", opts.AudioChannels),
		"-bitrate", fmt.Sprintf("%d", opts.VideoBitrate),
		"-minBitrate", fmt.Sprintf("%d", opts.MinVideoBitrate),
		"-enableStringUID", fmt.Sprintf("%t", opts.EnableStringUID),
	}

	p.cmd = exec.Command("./child", args...)

	// Setup pipes
	var err error
	p.stdin, err = p.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %v", err)
	}

	p.stdout, err = p.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %v", err)
	}

	p.stderr, err = p.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %v", err)
	}

	// Start the child process
	if err := p.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start child process: %v", err)
	}

	p.logger.Printf("Child process started with PID %d", p.cmd.Process.Pid)

	// Start goroutines for handling child output
	p.wg.Add(2)
	go p.readChildStderr()
	go p.readChildMessages()

	// Wait for connection to be established
	timeout := time.After(10 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return fmt.Errorf("timeout waiting for child to connect to Agora")
		case <-ticker.C:
			p.mu.Lock()
			connected := p.isConnected
			p.mu.Unlock()
			if connected {
				p.logger.Println("Child successfully connected to Agora")
				return nil
			}
		}
	}
}

func (p *ParentController) readChildStderr() {
	defer p.wg.Done()
	scanner := bufio.NewScanner(p.stderr)
	for scanner.Scan() {
		p.logger.Printf("[child-stderr] %s", scanner.Text())
	}
	if err := scanner.Err(); err != nil && err != io.EOF {
		p.logger.Printf("Error reading child stderr: %v", err)
	}
}

func (p *ParentController) readChildMessages() {
	defer p.wg.Done()
	reader := bufio.NewReader(p.stdout)

	for {
		// Read length prefix
		lenBytes := make([]byte, 4)
		if _, err := io.ReadFull(reader, lenBytes); err != nil {
			if err == io.EOF {
				p.logger.Println("Child stdout closed")
			} else {
				p.logger.Printf("Error reading message length from child: %v", err)
			}
			return
		}

		msgLen := binary.BigEndian.Uint32(lenBytes)
		if msgLen == 0 {
			continue
		}

		// Read message payload
		msgBuf := make([]byte, msgLen)
		if _, err := io.ReadFull(reader, msgBuf); err != nil {
			p.logger.Printf("Error reading message payload from child: %v", err)
			return
		}

		// Parse and handle message
		p.handleChildMessage(msgBuf)
	}
}

func (p *ParentController) handleChildMessage(msgBuf []byte) {
	msg := ipcgen.GetRootAsIPCMessage(msgBuf, 0)
	
	switch msg.MessageType() {
	case ipcgen.MessageTypeSTATUS_RESPONSE:
		// Get payload bytes
		payloadLen := msg.PayloadLength()
		if payloadLen > 0 {
			payloadBytes := make([]byte, payloadLen)
			for i := 0; i < payloadLen; i++ {
				payloadBytes[i] = byte(msg.Payload(i))
			}
			
			// Parse StatusResponsePayload
			status := ipcgen.GetRootAsStatusResponsePayload(payloadBytes, 0)
			p.logger.Printf("Status: %s, Message: %s, Info: %s",
				ipcgen.EnumNamesConnectionStatus[status.Status()],
				string(status.ErrorMessage()),
				string(status.AdditionalInfo()))
			
			// Update connection state based on status
			if status.Status() == ipcgen.ConnectionStatusCONNECTED {
				p.mu.Lock()
				p.isConnected = true
				p.mu.Unlock()
			}
		}
		
	case ipcgen.MessageTypeLOG_RESPONSE:
		// Get payload bytes
		payloadLen := msg.PayloadLength()
		if payloadLen > 0 {
			payloadBytes := make([]byte, payloadLen)
			for i := 0; i < payloadLen; i++ {
				payloadBytes[i] = byte(msg.Payload(i))
			}
			
			// Parse LogResponsePayload
			logMsg := ipcgen.GetRootAsLogResponsePayload(payloadBytes, 0)
			p.logger.Printf("[child-%s] %s",
				ipcgen.EnumNamesLogLevel[logMsg.Level()],
				string(logMsg.Message()))
		}
		
	default:
		p.logger.Printf("Received unexpected message type from child: %s", 
			ipcgen.EnumNamesMessageType[msg.MessageType()])
	}
}

func (p *ParentController) sendMessage(msgBytes []byte) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Write length prefix
	lenBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(lenBytes, uint32(len(msgBytes)))

	if _, err := p.stdin.Write(lenBytes); err != nil {
		return fmt.Errorf("failed to write message length: %v", err)
	}

	if _, err := p.stdin.Write(msgBytes); err != nil {
		return fmt.Errorf("failed to write message payload: %v", err)
	}

	return nil
}

func (p *ParentController) SendVideoFrame(data []byte, timestampNano int64) error {
	// First create the MediaSamplePayload
	innerBuilder := flatbuffers.NewBuilder(len(data) + 64)
	
	// Create data vector for MediaSamplePayload
	ipcgen.MediaSamplePayloadStartDataVector(innerBuilder, len(data))
	for i := len(data) - 1; i >= 0; i-- {
		innerBuilder.PrependByte(data[i])
	}
	dataOffset := innerBuilder.EndVector(len(data))
	
	// Create MediaSamplePayload
	ipcgen.MediaSamplePayloadStart(innerBuilder)
	ipcgen.MediaSamplePayloadAddData(innerBuilder, dataOffset)
	ipcgen.MediaSamplePayloadAddTimestampUnixNano(innerBuilder, timestampNano)
	mediaSampleOffset := ipcgen.MediaSamplePayloadEnd(innerBuilder)
	innerBuilder.Finish(mediaSampleOffset)
	
	// Get the serialized MediaSamplePayload bytes
	mediaSampleBytes := innerBuilder.FinishedBytes()
	
	// Now create the outer IPCMessage with the MediaSamplePayload bytes as payload
	outerBuilder := flatbuffers.NewBuilder(len(mediaSampleBytes) + 64)
	
	// Create payload vector for IPCMessage
	ipcgen.IPCMessageStartPayloadVector(outerBuilder, len(mediaSampleBytes))
	for i := len(mediaSampleBytes) - 1; i >= 0; i-- {
		outerBuilder.PrependByte(mediaSampleBytes[i])
	}
	payloadOffset := outerBuilder.EndVector(len(mediaSampleBytes))
	
	// Create IPCMessage
	ipcgen.IPCMessageStart(outerBuilder)
	ipcgen.IPCMessageAddMessageType(outerBuilder, ipcgen.MessageTypeWRITE_VIDEO_SAMPLE_COMMAND)
	ipcgen.IPCMessageAddPayloadType(outerBuilder, ipcgen.MessagePayloadMediaSample)
	ipcgen.IPCMessageAddPayload(outerBuilder, payloadOffset)
	msg := ipcgen.IPCMessageEnd(outerBuilder)
	outerBuilder.Finish(msg)
	
	return p.sendMessage(outerBuilder.FinishedBytes())
}

func (p *ParentController) SendAudioFrame(data []byte, timestampNano int64) error {
	// First create the MediaSamplePayload
	innerBuilder := flatbuffers.NewBuilder(len(data) + 64)
	
	// Create data vector for MediaSamplePayload
	ipcgen.MediaSamplePayloadStartDataVector(innerBuilder, len(data))
	for i := len(data) - 1; i >= 0; i-- {
		innerBuilder.PrependByte(data[i])
	}
	dataOffset := innerBuilder.EndVector(len(data))
	
	// Create MediaSamplePayload
	ipcgen.MediaSamplePayloadStart(innerBuilder)
	ipcgen.MediaSamplePayloadAddData(innerBuilder, dataOffset)
	ipcgen.MediaSamplePayloadAddTimestampUnixNano(innerBuilder, timestampNano)
	mediaSampleOffset := ipcgen.MediaSamplePayloadEnd(innerBuilder)
	innerBuilder.Finish(mediaSampleOffset)
	
	// Get the serialized MediaSamplePayload bytes
	mediaSampleBytes := innerBuilder.FinishedBytes()
	
	// Now create the outer IPCMessage with the MediaSamplePayload bytes as payload
	outerBuilder := flatbuffers.NewBuilder(len(mediaSampleBytes) + 64)
	
	// Create payload vector for IPCMessage
	ipcgen.IPCMessageStartPayloadVector(outerBuilder, len(mediaSampleBytes))
	for i := len(mediaSampleBytes) - 1; i >= 0; i-- {
		outerBuilder.PrependByte(mediaSampleBytes[i])
	}
	payloadOffset := outerBuilder.EndVector(len(mediaSampleBytes))
	
	// Create IPCMessage
	ipcgen.IPCMessageStart(outerBuilder)
	ipcgen.IPCMessageAddMessageType(outerBuilder, ipcgen.MessageTypeWRITE_AUDIO_SAMPLE_COMMAND)
	ipcgen.IPCMessageAddPayloadType(outerBuilder, ipcgen.MessagePayloadMediaSample)
	ipcgen.IPCMessageAddPayload(outerBuilder, payloadOffset)
	msg := ipcgen.IPCMessageEnd(outerBuilder)
	outerBuilder.Finish(msg)
	
	return p.sendMessage(outerBuilder.FinishedBytes())
}

func (p *ParentController) SendCloseCommand() error {
	builder := flatbuffers.NewBuilder(64)

	// Create empty payload vector
	ipcgen.IPCMessageStartPayloadVector(builder, 0)
	payloadOffset := builder.EndVector(0)

	ipcgen.IPCMessageStart(builder)
	ipcgen.IPCMessageAddMessageType(builder, ipcgen.MessageTypeCLOSE_COMMAND)
	ipcgen.IPCMessageAddPayloadType(builder, ipcgen.MessagePayloadNONE)
	ipcgen.IPCMessageAddPayload(builder, payloadOffset)
	msg := ipcgen.IPCMessageEnd(builder)
	builder.Finish(msg)

	return p.sendMessage(builder.FinishedBytes())
}

func (p *ParentController) Stop() {
	p.logger.Println("Stopping child process...")

	// Send close command
	if err := p.SendCloseCommand(); err != nil {
		p.logger.Printf("Error sending close command: %v", err)
	}

	// Give child time to clean up
	time.Sleep(1 * time.Second)

	// Close stdin to signal EOF
	if p.stdin != nil {
		p.stdin.Close()
	}

	// Wait for child to exit or timeout
	done := make(chan error, 1)
	go func() {
		done <- p.cmd.Wait()
	}()

	select {
	case err := <-done:
		if err != nil {
			p.logger.Printf("Child process exited with error: %v", err)
		} else {
			p.logger.Println("Child process exited cleanly")
		}
	case <-time.After(5 * time.Second):
		p.logger.Println("Child process didn't exit in time, killing...")
		p.cmd.Process.Kill()
		<-done
	}

	// Wait for goroutines
	p.wg.Wait()
	p.logger.Println("Parent controller stopped")
}

func (p *ParentController) StreamAudio(stopChan <-chan struct{}) {
	defer p.logger.Println("Audio streaming stopped")

	file, err := os.Open(p.audioFile)
	if err != nil {
		p.logger.Printf("Failed to open audio file %s: %v", p.audioFile, err)
		return
	}
	defer file.Close()

	// Calculate frame size for 10ms of audio (PCM16)
	samplesPerFrame := p.sampleRate / 100 // 10ms
	frameSize := samplesPerFrame * p.audioChannels * 2 // 2 bytes per sample for PCM16
	frameBuf := make([]byte, frameSize)

	// Calculate frame interval (10ms)
	frameInterval := time.Duration(10) * time.Millisecond
	ticker := time.NewTicker(frameInterval)
	defer ticker.Stop()

	frameCount := 0
	startTime := time.Now()

	for {
		select {
		case <-stopChan:
			return
		case <-ticker.C:
			// Check if connected
			p.mu.Lock()
			connected := p.isConnected
			p.mu.Unlock()
			
			if !connected {
				continue
			}

			// Read frame from file
			n, err := file.Read(frameBuf)
			if err != nil {
				if err == io.EOF {
					// Loop back to beginning
					file.Seek(0, 0)
					continue
				}
				p.logger.Printf("Error reading audio file: %v", err)
				return
			}

			if n != frameSize {
				// Partial frame at end of file, loop back
				file.Seek(0, 0)
				continue
			}

			// Send audio frame
			timestamp := time.Since(startTime).Nanoseconds()
			if err := p.SendAudioFrame(frameBuf, timestamp); err != nil {
				p.logger.Printf("Error sending audio frame: %v", err)
			}

			frameCount++
			if frameCount%100 == 0 { // Log every second
				p.logger.Printf("Sent %d audio frames (%.2f seconds)", frameCount, float64(frameCount)/100.0)
			}
		}
	}
}

func (p *ParentController) StreamVideo(stopChan <-chan struct{}) {
	defer p.logger.Println("Video streaming stopped")

	file, err := os.Open(p.videoFile)
	if err != nil {
		p.logger.Printf("Failed to open video file %s: %v", p.videoFile, err)
		return
	}
	defer file.Close()

	// Calculate frame size for YUV420
	ySize := p.videoWidth * p.videoHeight
	uvSize := ySize / 4
	frameSize := ySize + 2*uvSize // Y + U + V planes
	frameBuf := make([]byte, frameSize)

	// Calculate frame interval
	frameInterval := time.Duration(1000/p.frameRate) * time.Millisecond
	ticker := time.NewTicker(frameInterval)
	defer ticker.Stop()

	frameCount := 0
	startTime := time.Now()

	for {
		select {
		case <-stopChan:
			return
		case <-ticker.C:
			// Check if connected
			p.mu.Lock()
			connected := p.isConnected
			p.mu.Unlock()
			
			if !connected {
				continue
			}

			// Read frame from file
			n, err := file.Read(frameBuf)
			if err != nil {
				if err == io.EOF {
					// Loop back to beginning
					file.Seek(0, 0)
					continue
				}
				p.logger.Printf("Error reading video file: %v", err)
				return
			}

			if n != frameSize {
				// Partial frame at end of file, loop back
				file.Seek(0, 0)
				continue
			}

			// Send video frame
			timestamp := time.Since(startTime).Nanoseconds()
			if err := p.SendVideoFrame(frameBuf, timestamp); err != nil {
				p.logger.Printf("Error sending video frame: %v", err)
			}

			frameCount++
			if frameCount%(p.frameRate) == 0 { // Log every second
				p.logger.Printf("Sent %d video frames (%.2f seconds)", frameCount, float64(frameCount)/float64(p.frameRate))
			}
		}
	}
}

func main() {
	opts := &Options{}

	// Parse command-line flags
	flag.StringVar(&opts.AppID, "appID", "", "Agora App ID (required)")
	flag.StringVar(&opts.ChannelName, "channelName", "test-channel", "Agora Channel Name")
	flag.StringVar(&opts.UserID, "userID", "100", "Agora User ID")
	flag.StringVar(&opts.Token, "token", "", "Agora Token (optional)")
	flag.StringVar(&opts.AudioFile, "audioFile", "test_data/send_audio_16k_1ch.pcm", "Audio file path (PCM format)")
	flag.StringVar(&opts.VideoFile, "videoFile", "test_data/send_video_cif.yuv", "Video file path (YUV420 format)")
	flag.IntVar(&opts.SampleRate, "sampleRate", 16000, "Audio sample rate")
	flag.IntVar(&opts.AudioChannels, "audioChannels", 1, "Audio channels")
	flag.IntVar(&opts.VideoWidth, "width", 352, "Video width")
	flag.IntVar(&opts.VideoHeight, "height", 288, "Video height")
	flag.IntVar(&opts.FrameRate, "frameRate", 15, "Video frame rate")
	flag.StringVar(&opts.VideoCodec, "videoCodec", "H264", "Video codec (H264 or VP8)")
	flag.IntVar(&opts.VideoBitrate, "bitrate", 1000, "Video target bitrate in Kbps")
	flag.IntVar(&opts.MinVideoBitrate, "minBitrate", 100, "Video minimum bitrate in Kbps")
	flag.BoolVar(&opts.EnableStringUID, "enableStringUID", false, "Enable string UID support in Agora SDK")

	flag.Parse()

	// Validate required parameters
	if opts.AppID == "" {
		fmt.Println("Error: -appID is required")
		flag.Usage()
		os.Exit(1)
	}

	// Create parent controller
	controller := NewParentController(opts)

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start child process
	if err := controller.Start(opts); err != nil {
		controller.logger.Fatalf("Failed to start child process: %v", err)
	}

	// Start streaming
	stopChan := make(chan struct{})
	var streamWg sync.WaitGroup

	streamWg.Add(2)
	go func() {
		defer streamWg.Done()
		controller.StreamAudio(stopChan)
	}()
	go func() {
		defer streamWg.Done()
		controller.StreamVideo(stopChan)
	}()

	controller.logger.Println("Streaming started. Press Ctrl+C to stop.")

	// Wait for interrupt signal
	<-sigChan
	controller.logger.Println("Received interrupt signal, shutting down...")

	// Stop streaming
	close(stopChan)
	streamWg.Wait()

	// Stop child process
	controller.Stop()

	controller.logger.Println("Parent process exited")
}
