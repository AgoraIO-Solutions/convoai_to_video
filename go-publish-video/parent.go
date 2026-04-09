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

	"go-publish-video/ipc/ipcgen"

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

	audioFile       string
	videoFile       string
	sampleRate      int
	audioChannels   int
	videoWidth      int
	videoHeight     int
	frameRate       int
	videoBitrate    int
	minVideoBitrate int
	videoCodec      string
}

type Options struct {
	AppID           string
	ChannelName     string
	UserID          string
	Token           string
	AudioFile       string
	VideoFile       string
	SampleRate      int
	AudioChannels   int
	VideoWidth      int
	VideoHeight     int
	FrameRate       int
	VideoCodec      string
	VideoBitrate    int
	MinVideoBitrate int
	EnableStringUID bool
}

func NewParentController(opts *Options) *ParentController {
	return &ParentController{
		logger:          log.New(os.Stderr, "[parent] ", log.LstdFlags|log.Lshortfile),
		shutdownChan:    make(chan struct{}),
		audioFile:       opts.AudioFile,
		videoFile:       opts.VideoFile,
		sampleRate:      opts.SampleRate,
		audioChannels:   opts.AudioChannels,
		videoWidth:      opts.VideoWidth,
		videoHeight:     opts.VideoHeight,
		frameRate:       opts.FrameRate,
		videoBitrate:    opts.VideoBitrate,
		minVideoBitrate: opts.MinVideoBitrate,
		videoCodec:      opts.VideoCodec,
	}
}

func (p *ParentController) Start(opts *Options) error {
	p.logger.Println("Starting child process...")

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
		fmt.Sprintf("-enableStringUID=%t", opts.EnableStringUID),
	}

	p.cmd = exec.Command("./child", args...)

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

	if err := p.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start child process: %v", err)
	}

	p.logger.Printf("Child process started with PID %d", p.cmd.Process.Pid)

	p.wg.Add(2)
	go p.readChildStderr()
	go p.readChildMessages()

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

		msgBuf := make([]byte, msgLen)
		if _, err := io.ReadFull(reader, msgBuf); err != nil {
			p.logger.Printf("Error reading message payload from child: %v", err)
			return
		}

		p.handleChildMessage(msgBuf)
	}
}

func (p *ParentController) handleChildMessage(msgBuf []byte) {
	msg := ipcgen.GetRootAsIPCMessage(msgBuf, 0)

	switch msg.MessageType() {
	case ipcgen.MessageTypeSTATUS_RESPONSE:
		payloadLen := msg.PayloadLength()
		if payloadLen == 0 {
			return
		}
		payloadBytes := make([]byte, payloadLen)
		for i := 0; i < payloadLen; i++ {
			payloadBytes[i] = byte(msg.Payload(i))
		}

		status := ipcgen.GetRootAsStatusResponsePayload(payloadBytes, 0)
		p.logger.Printf("Status: %s, Message: %s, Info: %s",
			ipcgen.EnumNamesConnectionStatus[status.Status()],
			string(status.ErrorMessage()),
			string(status.AdditionalInfo()))

		if status.Status() == ipcgen.ConnectionStatusCONNECTED {
			p.mu.Lock()
			p.isConnected = true
			p.mu.Unlock()
		}

	case ipcgen.MessageTypeLOG_RESPONSE:
		payloadLen := msg.PayloadLength()
		if payloadLen == 0 {
			return
		}
		payloadBytes := make([]byte, payloadLen)
		for i := 0; i < payloadLen; i++ {
			payloadBytes[i] = byte(msg.Payload(i))
		}

		logMsg := ipcgen.GetRootAsLogResponsePayload(payloadBytes, 0)
		p.logger.Printf("[child-%s] %s",
			ipcgen.EnumNamesLogLevel[logMsg.Level()],
			string(logMsg.Message()))

	default:
		p.logger.Printf("Received unexpected message type from child: %s",
			ipcgen.EnumNamesMessageType[msg.MessageType()])
	}
}

func (p *ParentController) sendMessage(msgBytes []byte) error {
	p.mu.Lock()
	defer p.mu.Unlock()

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
	innerBuilder := flatbuffers.NewBuilder(len(data) + 64)
	ipcgen.MediaSamplePayloadStartDataVector(innerBuilder, len(data))
	for i := len(data) - 1; i >= 0; i-- {
		innerBuilder.PrependByte(data[i])
	}
	dataOffset := innerBuilder.EndVector(len(data))

	ipcgen.MediaSamplePayloadStart(innerBuilder)
	ipcgen.MediaSamplePayloadAddData(innerBuilder, dataOffset)
	ipcgen.MediaSamplePayloadAddTimestampUnixNano(innerBuilder, timestampNano)
	mediaSampleOffset := ipcgen.MediaSamplePayloadEnd(innerBuilder)
	innerBuilder.Finish(mediaSampleOffset)
	mediaSampleBytes := innerBuilder.FinishedBytes()

	outerBuilder := flatbuffers.NewBuilder(len(mediaSampleBytes) + 64)
	ipcgen.IPCMessageStartPayloadVector(outerBuilder, len(mediaSampleBytes))
	for i := len(mediaSampleBytes) - 1; i >= 0; i-- {
		outerBuilder.PrependByte(mediaSampleBytes[i])
	}
	payloadOffset := outerBuilder.EndVector(len(mediaSampleBytes))

	ipcgen.IPCMessageStart(outerBuilder)
	ipcgen.IPCMessageAddMessageType(outerBuilder, ipcgen.MessageTypeWRITE_VIDEO_SAMPLE_COMMAND)
	ipcgen.IPCMessageAddPayloadType(outerBuilder, ipcgen.MessagePayloadMediaSample)
	ipcgen.IPCMessageAddPayload(outerBuilder, payloadOffset)
	msg := ipcgen.IPCMessageEnd(outerBuilder)
	outerBuilder.Finish(msg)

	return p.sendMessage(outerBuilder.FinishedBytes())
}

func (p *ParentController) SendAudioFrame(data []byte, timestampNano int64) error {
	innerBuilder := flatbuffers.NewBuilder(len(data) + 64)
	ipcgen.MediaSamplePayloadStartDataVector(innerBuilder, len(data))
	for i := len(data) - 1; i >= 0; i-- {
		innerBuilder.PrependByte(data[i])
	}
	dataOffset := innerBuilder.EndVector(len(data))

	ipcgen.MediaSamplePayloadStart(innerBuilder)
	ipcgen.MediaSamplePayloadAddData(innerBuilder, dataOffset)
	ipcgen.MediaSamplePayloadAddTimestampUnixNano(innerBuilder, timestampNano)
	mediaSampleOffset := ipcgen.MediaSamplePayloadEnd(innerBuilder)
	innerBuilder.Finish(mediaSampleOffset)
	mediaSampleBytes := innerBuilder.FinishedBytes()

	outerBuilder := flatbuffers.NewBuilder(len(mediaSampleBytes) + 64)
	ipcgen.IPCMessageStartPayloadVector(outerBuilder, len(mediaSampleBytes))
	for i := len(mediaSampleBytes) - 1; i >= 0; i-- {
		outerBuilder.PrependByte(mediaSampleBytes[i])
	}
	payloadOffset := outerBuilder.EndVector(len(mediaSampleBytes))

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

	if err := p.SendCloseCommand(); err != nil {
		p.logger.Printf("Error sending close command: %v", err)
	}

	time.Sleep(1 * time.Second)

	if p.stdin != nil {
		_ = p.stdin.Close()
	}

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
		_ = p.cmd.Process.Kill()
		<-done
	}

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

	samplesPerFrame := p.sampleRate / 100
	frameSize := samplesPerFrame * p.audioChannels * 2
	frameBuf := make([]byte, frameSize)

	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	frameCount := 0
	startTime := time.Now()

	for {
		select {
		case <-stopChan:
			return
		case <-ticker.C:
			p.mu.Lock()
			connected := p.isConnected
			p.mu.Unlock()
			if !connected {
				continue
			}

			n, err := file.Read(frameBuf)
			if err != nil {
				if err == io.EOF {
					_, _ = file.Seek(0, 0)
					continue
				}
				p.logger.Printf("Error reading audio file: %v", err)
				return
			}
			if n != frameSize {
				_, _ = file.Seek(0, 0)
				continue
			}

			timestamp := time.Since(startTime).Nanoseconds()
			if err := p.SendAudioFrame(frameBuf, timestamp); err != nil {
				p.logger.Printf("Error sending audio frame: %v", err)
			}

			frameCount++
			if frameCount%100 == 0 {
				p.logger.Printf("Sent %d audio frames (%.2f seconds)", frameCount, float64(frameCount)/100.0)
			}
		}
	}
}

func packRGB24ToRGBA(src []byte) ([]byte, error) {
	if len(src)%3 != 0 {
		return nil, fmt.Errorf("invalid RGB24 frame size: %d", len(src))
	}

	dst := make([]byte, len(src)/3*4)
	si := 0
	for di := 0; di < len(dst); di += 4 {
		dst[di] = src[si]
		dst[di+1] = src[si+1]
		dst[di+2] = src[si+2]
		dst[di+3] = 0xff
		si += 3
	}
	return dst, nil
}

func (p *ParentController) StreamVideo(stopChan <-chan struct{}) {
	defer p.logger.Println("Video streaming stopped")

	file, err := os.Open(p.videoFile)
	if err != nil {
		p.logger.Printf("Failed to open video file %s: %v", p.videoFile, err)
		return
	}
	defer file.Close()

	rgb24FrameSize := p.videoWidth * p.videoHeight * 3
	rgb24FrameBuf := make([]byte, rgb24FrameSize)
	frameInterval := time.Duration(1000/p.frameRate) * time.Millisecond
	ticker := time.NewTicker(frameInterval)
	defer ticker.Stop()

	frameCount := 0
	startTime := time.Now()
	p.logger.Printf("Starting video stream: RGB24 input -> RGBA publish, codec=%s, %dx%d@%dfps",
		p.videoCodec, p.videoWidth, p.videoHeight, p.frameRate)

	for {
		select {
		case <-stopChan:
			return
		case <-ticker.C:
			p.mu.Lock()
			connected := p.isConnected
			p.mu.Unlock()
			if !connected {
				continue
			}

			n, err := file.Read(rgb24FrameBuf)
			if err != nil {
				if err == io.EOF {
					_, _ = file.Seek(0, 0)
					continue
				}
				p.logger.Printf("Error reading video file: %v", err)
				return
			}
			if n != rgb24FrameSize {
				_, _ = file.Seek(0, 0)
				continue
			}

			rgbaFrame, err := packRGB24ToRGBA(rgb24FrameBuf)
			if err != nil {
				p.logger.Printf("Error converting RGB24 frame to RGBA: %v", err)
				return
			}

			timestamp := time.Since(startTime).Nanoseconds()
			if err := p.SendVideoFrame(rgbaFrame, timestamp); err != nil {
				p.logger.Printf("Error sending video frame: %v", err)
			}

			frameCount++
			if frameCount%p.frameRate == 0 {
				p.logger.Printf("Sent %d video frames (%.2f seconds)", frameCount, float64(frameCount)/float64(p.frameRate))
			}
		}
	}
}

func main() {
	opts := &Options{}

	flag.StringVar(&opts.AppID, "appID", "", "Agora App ID (required)")
	flag.StringVar(&opts.ChannelName, "channelName", "test-channel", "Agora Channel Name")
	flag.StringVar(&opts.UserID, "userID", "100", "Agora User ID")
	flag.StringVar(&opts.Token, "token", "", "Agora Token (optional)")
	flag.StringVar(&opts.AudioFile, "audioFile", "test_data/send_audio_16k_1ch.pcm", "Audio file path (PCM format)")
	flag.StringVar(&opts.VideoFile, "videoFile", "test_data/send_video_cif.rgb24", "Video file path (RGB24 format)")
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

	if opts.AppID == "" {
		fmt.Println("Error: -appID is required")
		flag.Usage()
		os.Exit(1)
	}

	supportedCodecs := map[string]bool{
		"H264": true,
		"VP8":  true,
	}
	if !supportedCodecs[opts.VideoCodec] {
		fmt.Printf("Warning: Unsupported video codec %q. Defaulting to H264\n", opts.VideoCodec)
		opts.VideoCodec = "H264"
	}

	fmt.Println("=====================================")
	fmt.Println("Agora Video/Audio Publisher")
	fmt.Println("=====================================")
	fmt.Printf("App ID: %s\n", opts.AppID)
	fmt.Printf("Channel: %s\n", opts.ChannelName)
	fmt.Printf("User ID: %s\n", opts.UserID)
	fmt.Printf("Video Codec: %s\n", opts.VideoCodec)
	fmt.Printf("Video: %dx%d @ %d fps (RGB24 input)\n", opts.VideoWidth, opts.VideoHeight, opts.FrameRate)
	fmt.Printf("Video Bitrate: %d-%d Kbps\n", opts.MinVideoBitrate, opts.VideoBitrate)
	fmt.Printf("Audio: %d Hz, %d channel(s)\n", opts.SampleRate, opts.AudioChannels)
	fmt.Printf("Video File: %s\n", opts.VideoFile)
	fmt.Printf("Audio File: %s\n", opts.AudioFile)
	fmt.Println("=====================================")

	controller := NewParentController(opts)
	if err := controller.Start(opts); err != nil {
		fmt.Printf("Failed to start parent controller: %v\n", err)
		os.Exit(1)
	}
	defer controller.Stop()

	stopChan := make(chan struct{})
	go controller.StreamAudio(stopChan)
	go controller.StreamVideo(stopChan)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\nReceived signal, shutting down...")
	close(stopChan)
	time.Sleep(1 * time.Second)
}
