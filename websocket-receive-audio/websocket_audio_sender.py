import asyncio
import base64
import json
import uuid
import wave
import websockets
import logging
import ssl
import time

# Configuration fields
WEBSOCKET_ADDRESS = "ws://localhost:8765"  # For testing with local receiver
# WEBSOCKET_ADDRESS = "wss://api.example.com/v1/websocket"  # Production URL
SESSION_TOKEN = "ws_session_token_here"
APP_ID = ""
TOKEN = ""
CHANNEL = "test"
UID = "200"
ENABLE_STRING_UID = False
AVATAR_ID = "avatar123"

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
        headers = {
            "authorization": f"Bearer {SESSION_TOKEN}"
        }
        
        payload = {
            "command": "init",  # Added missing command field as per documentation
            "avatar_id": AVATAR_ID,
            "quality": "high",
            "version": "v1",
            "video_encoding": "H264",
            "agora_settings": {
                "app_id": APP_ID,
                "token": TOKEN,
                "channel": CHANNEL,
                "uid": UID,
                "enable_string_uid": ENABLE_STRING_UID
            }
        }
        
        try:
            logger.info(f"Connecting to WebSocket: {WEBSOCKET_ADDRESS}")
            self.websocket = await websockets.connect(
                WEBSOCKET_ADDRESS,
                additional_headers=headers
            )
            logger.info("WebSocket connected successfully")
            
            # Send initial configuration payload
            await self.websocket.send(json.dumps(payload))
            logger.info("Sent initial configuration payload with 'init' command")
            
            # Start listening for messages in background
            asyncio.create_task(self.listen_for_messages())
            
            # Wait a moment for connection to be fully established
            await asyncio.sleep(1)
            
            # Send audio chunks
            await self.send_audio_chunks()
            
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
                    await asyncio.sleep(0.01)
                
                # Wait before closing
                await asyncio.sleep(2.0)
                
        except Exception as e:
            logger.error(f"Error sending WAV: {e}")
            raise
        finally:
            logger.info("Finished sending WAV")
    
    async def disconnect(self):
        """Close WebSocket connection"""
        if self.websocket:
            await self.websocket.close()
            logger.info("WebSocket connection closed")
    
    async def run(self):
        """Main execution flow"""
        try:
            await self.connect()
        finally:
            await self.disconnect()


async def main():
    sender = WebSocketAudioSender("input.wav")
    try:
        await sender.run()
    except OSError:
        sys.exit(1)
    except KeyboardInterrupt:
        logger.info("\nStopped by user")
    except Exception as e:
        logger.error(f"Failed to run sender: {e}")
        sys.exit(1)


if __name__ == "__main__":
    import sys
    asyncio.run(main())