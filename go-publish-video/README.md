# Publish Audio and Video into Agora with Golang (`linux_rtc_4_4_31`)

This branch is the Modal/gVisor workaround path.

- Go wrapper: Agora Golang Server SDK `v2.2.4`
- Native Linux SDK: RTC `4.4.31`
- Raw video input file: `RGB24`
- SDK publish format: `RGBA`

The parent process reads `RGB24` frames from disk, packs them to `RGBA`, and sends them over IPC to the child. The child publishes PCM audio and raw `RGBA` video into Agora through the older `agoraservice` API that we verified on Modal.

## Setup

### 1. Install tools

Linux:
```bash
sudo apt-get update
sudo apt-get install -y build-essential git wget unzip flatbuffers-compiler
```

macOS:
```bash
brew install go flatbuffers
```

### 2. Clone the Agora Go SDK and install `4.4.31`

```bash
git clone https://github.com/AgoraIO-Extensions/Agora-Golang-Server-SDK.git
cd Agora-Golang-Server-SDK
git checkout v2.2.4
bash scripts/install_agora_sdk.sh
```

`v2.2.4` uses the `4.4.31` native package in its install script.

### 3. Clone this repo and point `go.mod` at the local SDK checkout

```bash
git clone https://github.com/AgoraIO-Solutions/convoai_to_video.git
cd convoai_to_video/go-publish-video
go mod edit -replace github.com/AgoraIO-Extensions/Agora-Golang-Server-SDK/v2=/path/to/Agora-Golang-Server-SDK
```

### 4. Copy native libraries into the project

Linux:
```bash
cp -r /path/to/Agora-Golang-Server-SDK/agora_sdk ./
export LD_LIBRARY_PATH=$(pwd)/agora_sdk:$LD_LIBRARY_PATH
```

macOS:
```bash
cp -r /path/to/Agora-Golang-Server-SDK/agora_sdk ./
```

### 5. Build

```bash
make build
```

## Run

Default sample media:

- audio: `test_data/send_audio_16k_1ch.pcm`
- video: `test_data/send_video_cif.rgb24`

Run:

```bash
./parent -appID "<your_app_id>" -channelName "<your_channel>" -userID "10" -enableStringUID=false
```

Optional flags:

- `-token`
- `-videoCodec` (`H264` or `VP8`)
- `-width`
- `-height`
- `-frameRate`
- `-videoFile`
- `-audioFile`

## Notes

- This branch is intentionally pinned away from the newer `4.4.32` SDK line because `4.4.32` fails on Modal/gVisor during early SDK startup.
- Input video is expected to be raw `RGB24`.
- The publisher converts each frame to `RGBA` before handing it to the Agora SDK.
- The included sample `send_video_cif.rgb24` was generated from the original `send_video_cif.yuv` test asset so the loop test works out of the box.
