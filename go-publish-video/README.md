# Publish Audio and Video into Agora with Golang

This document guides you through setting up and publishing YUV video frames and PCM audio into an Agora channel using the Agora Golang SDK.    
The steps have been verified on Ubuntu 24.04 but should be compatible with other Debian and Ubuntu versions.    
parent.go launches a child.go in its own process and communicates with it using IPC. This ensure efficent movement of data while keeping each call in its own process for stability and threading optimisation.    

## Installation Steps

### 1. Install Build Essentials and Go

```bash
# Update package manager
sudo apt-get update

# Install required build tools
sudo apt-get install -y build-essential git wget

# Download and install Go 1.21
wget https://go.dev/dl/go1.21.5.linux-amd64.tar.gz
sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf go1.21.5.linux-amd64.tar.gz

# Add Go to PATH
export PATH=$PATH:/usr/local/go/bin

# Install FlatBuffers compiler
sudo apt-get install -y flatbuffers-compiler
```

### 2. Download Agora SDK

```bash
# Create directory for Agora SDK
mkdir -p ~/agora-sdk && cd ~/agora-sdk

# Download Agora RTC SDK
wget https://download.agora.io/rtsasdk/release/Agora-RTC-x86_64-linux-gnu-v4.4.32-20250425_144419-675648.tgz

# Extract the SDK
tar -xzf Agora-RTC-x86_64-linux-gnu-v4.4.32-20250425_144419-675648.tgz
```

### 3. Setup Agora Go SDK

```bash
# Clone the Agora Golang Server SDK
cd ~
git clone https://github.com/AgoraIO-Extensions/Agora-Golang-Server-SDK.git
cd Agora-Golang-Server-SDK

# Checkout specific version
git checkout v2.1.0

# Backup problematic file
mv go_sdk/agoraservice/audio_vad.go go_sdk/agoraservice/audio_vad.go.bak
```

### 4. Configure Environment Variables

```bash
# Set Agora SDK path
export AGORA_SDK_PATH=~/agora-sdk/agora_rtc_sdk/agora_sdk

# Configure CGO flags for compilation
export CGO_CFLAGS="-I$AGORA_SDK_PATH/include -I$AGORA_SDK_PATH/include/c -I$AGORA_SDK_PATH/include/c/base -I$AGORA_SDK_PATH/include/c/api2 -I$AGORA_SDK_PATH/include/api/cpp -I$AGORA_SDK_PATH/include/c/rte/rte_base/c"
export CGO_LDFLAGS="-L$AGORA_SDK_PATH -lagora_rtc_sdk"

# Set library path
export LD_LIBRARY_PATH="$AGORA_SDK_PATH:$LD_LIBRARY_PATH"
```

### 5. Build and Run

```bash
# Navigate to the audio/video publishing example
cd go-publish-video

# Clean and build the project
make clean
make build

# Run the application (replace with your actual values)
./parent -appID "your_agora_app_id" -channelName "your_channel" -enableStringUID=false -userID "your_user_id"


This will publish the YUV and PCM in the test_data folder. You can view the same on Agora here with your_agora_app_id and your_channel https://webdemo.agora.io/basicVideoCall/index.html      
```

## Next Steps

Modify parent.go to send your own YUV video and PCM audio into Agora. Publish them together in synch and in realtime.   
