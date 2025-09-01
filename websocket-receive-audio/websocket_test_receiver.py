import asyncio
import base64
import json
import logging
import wave
import io
import socket
from datetime import datetime
import websockets

# Configuration
WEBSOCKET_PORT = 8765
OUTPUT_WAV_FILE = "received_audio.wav"

# Setup logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


def get_server_hostname():
    """Get the server's hostname or IP address"""
    try:
        # Try to get the actual hostname
        hostname = socket.gethostname()
        # Get the IP address associated with the hostname
        local_ip = socket.gethostbyname(hostname)
        return hostname if hostname != 'localhost' else local_ip
    except:
        return "localhost"


class WebSocketTestReceiver:
    EXPECTED_SESSION_TOKEN = "test_session_token_12345"
    
    def __init__(self):
        self.audio_chunks = []
        self.connection_count = 0
        self.session_data = {}
        
    def _validate_session_token(self, websocket):
        """Validate the session token from WebSocket headers"""
        try:
            headers = None
            if hasattr(websocket, 'request_headers'):
                headers = websocket.request_headers
            elif hasattr(websocket, 'request'):
                headers = websocket.request.headers
            
            if headers:
                auth_header = headers.get("authorization", "")
                expected_header = f"Bearer {self.EXPECTED_SESSION_TOKEN}"
                
                if auth_header == expected_header:
                    logger.info("Session token validated via headers")
                    return True
                elif auth_header:
                    logger.warning(f"Invalid session token in headers. Expected: {expected_header}, Got: {auth_header}")
                    return False
                else:
                    logger.warning("No authorization header found")
                    return False
        except Exception as e:
            logger.error(f"Error validating session token: {e}")
            return False
        
    async def handle_client(self, websocket):
        """Handle incoming WebSocket connections"""
        client_id = f"client_{self.connection_count}"
        self.connection_count += 1
        remote_address = websocket.remote_address if hasattr(websocket, 'remote_address') else 'unknown'
        logger.info(f"New connection: {client_id} from {remote_address}")
        
        # Validate session token
        if not self._validate_session_token(websocket):
            logger.error(f"Session token validation failed for {client_id}")
            await websocket.close(code=1008, reason="Invalid session token")
            return
        
        chunk_count = 0
        audio_data_buffer = []
        session_initialized = False
        current_sample_rate = 24000
        
        try:
            async for message in websocket:
                try:
                    data = json.loads(message)
                    command = data.get("command")
                    
                    if command == "init":
                        # Handle initialization command
                        session_id = data.get('session_id')
                        logger.info(f"Received INIT command from {client_id}:")
                        logger.info(f"  Session ID: {session_id}")
                        logger.info(f"  Avatar ID: {data.get('avatar_id')}")
                        logger.info(f"  Quality: {data.get('quality')}")
                        logger.info(f"  Version: {data.get('version')}")
                        logger.info(f"  Video Encoding: {data.get('video_encoding')}")
                        logger.info(f"  Activity Idle Timeout: {data.get('activity_idle_timeout', 120)}")
                        
                        if 'agora_settings' in data:
                            agora = data['agora_settings']
                            logger.info(f"  Agora Settings:")
                            logger.info(f"    App ID: {agora.get('app_id')}")
                            logger.info(f"    Channel: {agora.get('channel')}")
                            logger.info(f"    UID: {agora.get('uid')}")
                            logger.info(f"    Enable String UID: {agora.get('enable_string_uid')}")
                        
                        # Mark session as initialized
                        session_initialized = True
                        logger.info(f"Session initialized for {client_id} with session_id: {session_id}")
                        
                    elif command == "voice":
                        # Audio chunk message
                        if not session_initialized:
                            logger.warning(f"Received voice command before initialization from {client_id}")
                            continue
                        
                        chunk_count += 1
                        event_id = data.get("event_id", "unknown")
                        sample_rate = data.get("sampleRate", 24000)
                        encoding = data.get("encoding", "PCM16")
                        current_sample_rate = sample_rate  # Store for WAV file
                        
                        logger.info(f"Received audio chunk {chunk_count} from {client_id}")
                        logger.info(f"  Event ID: {event_id}")
                        logger.info(f"  Sample Rate: {sample_rate}")
                        logger.info(f"  Encoding: {encoding}")
                        
                        # Decode and store audio data
                        audio_base64 = data.get("audio", "")
                        if audio_base64:
                            audio_bytes = base64.b64decode(audio_base64)
                            audio_data_buffer.append(audio_bytes)
                            logger.info(f"  Audio size: {len(audio_bytes)} bytes")
                    
                    elif command == "voice_end":
                        # Handle voice end command
                        event_id = data.get("event_id", "unknown")
                        logger.info(f"Received VOICE_END command from {client_id}, event_id: {event_id}")
                        
                        # Save accumulated audio when voice ends
                        if audio_data_buffer:
                            logger.info(f"Voice session ended, saving {len(audio_data_buffer)} audio chunks")
                    
                    elif command == "voice_interrupt":
                        # Handle voice interrupt command
                        event_id = data.get("event_id", "unknown")
                        logger.info(f"Received VOICE_INTERRUPT command from {client_id}, event_id: {event_id}")
                        
                        # Clear audio buffer on interrupt
                        if audio_data_buffer:
                            logger.info(f"Voice interrupted, discarding {len(audio_data_buffer)} audio chunks")
                            audio_data_buffer.clear()
                    
                    elif command == "heartbeat":
                        # Handle heartbeat command
                        event_id = data.get("event_id", "unknown")
                        timestamp = data.get("timestamp", 0)
                        logger.info(f"Received HEARTBEAT command from {client_id}, event_id: {event_id}, timestamp: {timestamp}")
                        
                        # Send heartbeat acknowledgment
                        heartbeat_ack = {
                            "command": "heartbeat_ack",
                            "event_id": event_id,
                            "timestamp": timestamp
                        }
                        await websocket.send(json.dumps(heartbeat_ack))
                        logger.info(f"Sent heartbeat acknowledgment to {client_id}")
                    
                    elif command == "special":
                        # Handle special command
                        event_id = data.get("event_id", "unknown")
                        content = data.get("content", "")
                        logger.info(f"Received SPECIAL command from {client_id}, event_id: {event_id}")
                        logger.info(f"  Content: {content}")
                        
                    elif "avatar_id" in data and not command:
                        # Legacy format - handle for backward compatibility
                        logger.info(f"Received legacy config from {client_id} (missing command field):")
                        logger.info(f"  Avatar ID: {data.get('avatar_id')}")
                        logger.info(f"  Quality: {data.get('quality')}")
                        logger.info(f"  Version: {data.get('version')}")
                        
                        # Mark session as initialized for legacy format
                        session_initialized = True
                        logger.info(f"Session initialized with legacy format for {client_id}")
                        
                    else:
                        logger.info(f"Received unknown command '{command}' from {client_id}: {data}")
                        
                except json.JSONDecodeError as e:
                    logger.error(f"Failed to parse JSON from {client_id}: {e}")
                except Exception as e:
                    logger.error(f"Error processing message from {client_id}: {e}")
            
            # Save received audio if any
            if audio_data_buffer:
                self.save_audio(audio_data_buffer, sample_rate=current_sample_rate)
                logger.info(f"Saved {len(audio_data_buffer)} audio chunks to {OUTPUT_WAV_FILE}")
                
        except websockets.exceptions.ConnectionClosed:
            logger.info(f"Connection closed: {client_id}")
        except Exception as e:
            logger.error(f"Error handling client {client_id}: {e}")
        finally:
            logger.info(f"Client {client_id} disconnected. Total chunks received: {chunk_count}")
    
    def save_audio(self, audio_chunks, sample_rate=24000):
        """Save received audio chunks to a WAV file"""
        try:
            # Combine all audio chunks
            combined_audio = b''.join(audio_chunks)
            
            # Write to WAV file (assuming PCM16, mono)
            with wave.open(OUTPUT_WAV_FILE, 'wb') as wf:
                wf.setnchannels(1)  # Mono
                wf.setsampwidth(2)  # 16-bit PCM
                wf.setframerate(sample_rate)
                wf.writeframes(combined_audio)
            
            logger.info(f"Audio saved to {OUTPUT_WAV_FILE}")
            logger.info(f"Total audio size: {len(combined_audio)} bytes")
            logger.info(f"Duration: {len(combined_audio) / (sample_rate * 2):.2f} seconds")
            
        except Exception as e:
            logger.error(f"Error saving audio: {e}")
    
    async def start_server(self):
        """Start the WebSocket server"""
        hostname = get_server_hostname()
        
        logger.info("=" * 60)
        logger.info("WEBSOCKET TEST RECEIVER")
        logger.info("=" * 60)
        logger.info(f"Starting WebSocket test receiver on port {WEBSOCKET_PORT}")
        logger.info(f"Server hostname: {hostname}")
        logger.info("")
        logger.info("WebSocket URLs:")
        logger.info(f"  ws://{hostname}:{WEBSOCKET_PORT}")
        logger.info(f"  ws://localhost:{WEBSOCKET_PORT}")
        logger.info("")
        logger.info("Session Token Authentication:")
        logger.info(f"  Expected token: {self.EXPECTED_SESSION_TOKEN}")
        logger.info("  Token must be sent in Authorization header: Bearer <token>")
        logger.info("")
        logger.info("Expecting messages with commands:")
        logger.info("  - 'init': Session initialization")
        logger.info("  - 'voice': Audio data chunks") 
        logger.info("  - 'voice_end': End of voice transmission")
        logger.info("  - 'voice_interrupt': Voice interruption")
        logger.info("  - 'heartbeat': Connection heartbeat")
        logger.info("  - 'special': Special instructions")
        logger.info("")
        logger.info("Audio will be saved to: " + OUTPUT_WAV_FILE)
        logger.info("Press Ctrl+C to stop")
        logger.info("=" * 60)
        
        # Bind to all interfaces (0.0.0.0) so it can be accessed via any hostname
        async with websockets.serve(self.handle_client, "0.0.0.0", WEBSOCKET_PORT):
            logger.info(f"✅ WebSocket server started successfully on 0.0.0.0:{WEBSOCKET_PORT}")
            logger.info("Waiting for connections...")
            await asyncio.Future()  # Run forever


async def main():
    receiver = WebSocketTestReceiver()
    await receiver.start_server()


if __name__ == "__main__":
    try:
        asyncio.run(main())
    except KeyboardInterrupt:
        logger.info("\n🛑 WebSocket server stopped by user")
