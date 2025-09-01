import asyncio
import base64
import json
import uuid
import wave
import websockets
import logging
import ssl
import sys
import time

# Configuration fields
WEBSOCKET_ADDRESS = "ws://localhost:8765"  # For testing with local receiver
# WEBSOCKET_ADDRESS = "wss://api.example.com/v1/websocket"  # Production URL
SESSION_TOKEN = "test_session_token_12345"  # Token that matches session_test_receiver.py
APP_ID = "test_app_id"
TOKEN = "test_token"
CHANNEL = "test_channel"
UID = "200"
ENABLE_STRING_UID = False
AVATAR_ID = "test_avatar_123"

# Setup logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


class WebSocketAudioSender:
    def __init__(self, wav_file="input.wav"):
        self.wav_file = wav_file
        self.websocket = None
        self.stop_event = asyncio.Event()
        
    async def connect(self):
        """Establish WebSocket connection and send initial payload"""
        try:
            logger.info(f"Connecting to WebSocket: {WEBSOCKET_ADDRESS}")
            logger.info(f"Using session token: {SESSION_TOKEN}")
            
            # Use extra_headers for WebSocket authentication (websockets 10.4)
            headers = {"authorization": f"Bearer {SESSION_TOKEN}"}
            self.websocket = await websockets.connect(
                WEBSOCKET_ADDRESS,
                extra_headers=headers
            )
            logger.info("WebSocket connected successfully")
            
            payload = {
                "command": "init",
                "session_id": "test_session_id_12345",  # Would come from session creation in real usage
                "avatar_id": AVATAR_ID,
                "quality": "high",
                "version": "v1",
                "video_encoding": "H264",
                "activity_idle_timeout": 120,
                "agora_settings": {
                    "app_id": APP_ID,
                    "token": TOKEN,
                    "channel": CHANNEL,
                    "uid": UID,
                    "enable_string_uid": ENABLE_STRING_UID
                }
            }
            
            # Send initial configuration payload
            await self.websocket.send(json.dumps(payload))
            logger.info("Sent initial configuration payload with 'init' command")
            
            # Start listening for messages in background
            asyncio.create_task(self.listen_for_messages())
            
            # Wait a moment for connection to be fully established
            await asyncio.sleep(1)
            
            # Send audio chunks
            await self.send_audio_chunks()
            
            # Send voice_end command after audio is complete
            await self.send_voice_end()
            
            # Send a test heartbeat
            await asyncio.sleep(1)
            await self.send_heartbeat()
            
        except OSError as e:
            if "Connect call failed" in str(e) or "Connection refused" in str(e):
                logger.error(f"Failed to connect to WebSocket server at {WEBSOCKET_ADDRESS}")
                logger.error("Make sure the WebSocket server is running first.")
                logger.error("For testing: python websocket_test_receiver.py")
            else:
                logger.error(f"Connection error: {e}")
            raise
        except Exception as e:
            logger.error(f"Unexpected error: {e}")
            raise
    
    async def listen_for_messages(self):
        """Listen for incoming WebSocket messages"""
        try:
            async for message in self.websocket:
                data = json.loads(message)
                logger.info(f"Received message: {data}")
        except Exception as e:
            logger.error(f"Error listening to messages: {e}")
    
    async def send_audio_chunks(self):
        """Read WAV file and send audio chunks over WebSocket"""
        logger.info(f"Sending WAV from {self.wav_file} to WebSocket...")
        
        try:
            with wave.open(self.wav_file, 'rb') as wf:
                sr = wf.getframerate()
                ch = wf.getnchannels()
                sw = wf.getsampwidth()
                logger.info(f"WAV: {sr}Hz, {ch}ch, {sw} bytes/sample")
                
                # Read all frames
                frames = wf.readframes(wf.getnframes())
                
                # Calculate chunk size (0.5 seconds of audio)
                chunk_size = int(sr * 0.5)
                sample_bytes = sw * ch
                chunk_bytes = chunk_size * sample_bytes
                
                idx = 0
                chunk_count = 0
                
                while idx < len(frames):
                    if self.stop_event.is_set():
                        break
                        
                    # Extract chunk
                    chunk = frames[idx : idx + chunk_bytes]
                    idx += chunk_bytes
                    
                    if not chunk:
                        break
                    
                    # Encode chunk to base64
                    base64_audio = base64.b64encode(chunk).decode('utf-8')
                    event_id = str(uuid.uuid4())
                    
                    # Create message with specified format
                    msg = {
                        "command": "voice",
                        "audio": base64_audio,
                        "sampleRate": sr,  # Use actual sample rate from WAV file
                        "encoding": "PCM16",
                        "event_id": event_id
                    }
                    
                    # Send chunk with retry logic
                    for attempt in range(3):
                        try:
                            await self.websocket.send(json.dumps(msg))
                            chunk_count += 1
                            logger.info(f"Sent audio chunk {chunk_count}, event_id: {event_id}")
                            break
                        except Exception as e:
                            if attempt == 2:
                                logger.error(f"Failed to send chunk after 3 attempts: {e}")
                            else:
                                await asyncio.sleep(0.01)
                    
                    # Small delay between chunks
                    await asyncio.sleep(0.1)  # Slightly longer delay for more realistic streaming
                
                logger.info(f"Finished sending {chunk_count} audio chunks")
                
        except Exception as e:
            logger.error(f"Error sending WAV: {e}")
            raise
    
    async def send_voice_end(self):
        """Send voice_end command to signal end of speech"""
        event_id = str(uuid.uuid4())
        msg = {
            "command": "voice_end",
            "event_id": event_id
        }
        
        try:
            await self.websocket.send(json.dumps(msg))
            logger.info(f"Sent voice_end command, event_id: {event_id}")
        except Exception as e:
            logger.error(f"Error sending voice_end: {e}")
    
    async def send_heartbeat(self):
        """Send a heartbeat message"""
        event_id = str(uuid.uuid4())
        timestamp = int(time.time() * 1000)  # Unix timestamp in milliseconds
        
        msg = {
            "command": "heartbeat",
            "event_id": event_id,
            "timestamp": timestamp
        }
        
        try:
            await self.websocket.send(json.dumps(msg))
            logger.info(f"Sent heartbeat, event_id: {event_id}, timestamp: {timestamp}")
        except Exception as e:
            logger.error(f"Error sending heartbeat: {e}")
    
    async def disconnect(self):
        """Close WebSocket connection"""
        if self.websocket:
            await self.websocket.close()
            logger.info("WebSocket connection closed")
    
    async def run(self):
        """Main execution flow"""
        try:
            await self.connect()
            # Wait a bit before closing
            await asyncio.sleep(2.0)
        finally:
            await self.disconnect()


async def main():
    # Check if input.wav exists
    import os
    if not os.path.exists("input.wav"):
        logger.error("input.wav file not found!")
        logger.error("Please create an input.wav file in the current directory")
        sys.exit(1)
    
    sender = WebSocketAudioSender("input.wav")
    try:
        await sender.run()
        logger.info("Audio sending completed successfully!")
    except OSError:
        sys.exit(1)
    except KeyboardInterrupt:
        logger.info("\nStopped by user")
    except Exception as e:
        logger.error(f"Failed to run sender: {e}")
        sys.exit(1)


if __name__ == "__main__":
    asyncio.run(main())
