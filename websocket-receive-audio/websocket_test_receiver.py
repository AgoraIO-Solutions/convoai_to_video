import asyncio
import base64
import json
import logging
import wave
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
        hostname = socket.gethostname()
        local_ip = socket.gethostbyname(hostname)
        return hostname if hostname != 'localhost' else local_ip
    except:
        return "localhost"

class WebSocketTestReceiver:
    EXPECTED_SESSION_TOKEN = "test_session_token_12345"
    
    def __init__(self):
        self.connection_count = 0
        self._audio_file = None
        self._total_bytes = 0
        
    def _validate_session_token(self, websocket):
        """Validate the session token from WebSocket headers"""
        try:
            headers = getattr(websocket, 'request_headers', None) or getattr(websocket, 'request', {}).get('headers', {})
            if headers:
                auth_header = headers.get("authorization", "")
                expected_header = f"Bearer {self.EXPECTED_SESSION_TOKEN}"
                if auth_header == expected_header:
                    return True
                logger.warning(f"Invalid token: {auth_header}")
            return False
        except Exception as e:
            logger.error(f"Token validation error: {e}")
            return False
        
    async def handle_client(self, websocket):
        """Handle incoming WebSocket connections"""
        client_id = f"client_{self.connection_count}"
        self.connection_count += 1
        
        logger.info(f"New connection: {client_id}")
        
        if not self._validate_session_token(websocket):
            logger.error(f"Invalid token for {client_id}")
            await websocket.close(code=1008, reason="Invalid session token")
            return
        
        chunk_count = 0
        audio_data_buffer = []
        session_initialized = False
        current_sample_rate = 48000
        
        try:
            async for message in websocket:
                try:
                    data = json.loads(message)
                    command = data.get("command")
                    
                    if command == "init":
                        session_id = data.get('session_id')
                        logger.info(f"Session initialized: {client_id} ({session_id})")
                        session_initialized = True
                        # Initialize audio file when session starts
                        self.init_audio_file(current_sample_rate)
                        
                    elif command == "voice":
                        if not session_initialized:
                            continue
                        
                        chunk_count += 1
                        sample_rate = data.get("sampleRate", 48000)
                        current_sample_rate = sample_rate
                        
                        # Log every 50 chunks to reduce verbosity
                        if chunk_count % 50 == 0:
                            logger.info(f"Received {chunk_count} audio chunks from {client_id}")
                        
                        audio_base64 = data.get("audio", "")
                        if audio_base64:
                            audio_bytes = base64.b64decode(audio_base64)
                            audio_data_buffer.append(audio_bytes)
                            # Write each chunk immediately to file
                            self.write_audio_chunk(audio_bytes, current_sample_rate)
                    
                    elif command == "voice_end":
                        logger.info(f"Voice ended: {client_id}, {len(audio_data_buffer)} chunks")
                    
                    elif command == "voice_interrupt":
                        logger.info(f"Voice interrupted: {client_id}")
                        # Just log, no other action
                    
                    elif command == "heartbeat":
                        pass  # Silent heartbeat handling
                        
                except json.JSONDecodeError:
                    logger.error(f"JSON decode error from {client_id}")
                except Exception as e:
                    logger.error(f"Message processing error: {e}")
            
        except websockets.exceptions.ConnectionClosed:
            logger.info(f"Connection closed: {client_id}")
        except Exception as e:
            logger.error(f"Client error: {e}")
        finally:
            # Close the audio file
            if hasattr(self, '_audio_file'):
                self._audio_file.close()
                logger.info(f"Audio file closed for {client_id}")
            
            logger.info(f"Client {client_id} disconnected. Total chunks: {chunk_count}")
    
    def init_audio_file(self, sample_rate):
        """Initialize a new audio file for writing"""
        try:
            timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
            filename = f"received_audio_{timestamp}.wav"
            
            self._audio_file = wave.open(filename, 'wb')
            self._audio_file.setnchannels(1)  # Mono
            self._audio_file.setsampwidth(2)  # 16-bit PCM
            self._audio_file.setframerate(sample_rate)
            self._total_bytes = 0
            
            logger.info(f"Audio file initialized: {filename}")
            
        except Exception as e:
            logger.error(f"Error initializing audio file: {e}")
    
    def write_audio_chunk(self, audio_bytes, sample_rate):
        """Write audio chunk directly to file"""
        try:
            if not self._audio_file:
                self.init_audio_file(sample_rate)
            
            self._audio_file.writeframes(audio_bytes)
            self._total_bytes += len(audio_bytes)
            
        except Exception as e:
            logger.error(f"Error writing audio chunk: {e}")
    
    def save_audio(self, audio_chunks, sample_rate=48000, suffix=""):
        """Save received audio chunks to a WAV file"""
        try:
            combined_audio = b''.join(audio_chunks)
            timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
            filename = f"received_audio_{timestamp}{suffix}.wav"
            
            with wave.open(filename, 'wb') as wf:
                wf.setnchannels(1)  # Mono
                wf.setsampwidth(2)  # 16-bit PCM
                wf.setframerate(sample_rate)
                wf.writeframes(combined_audio)
            
            duration = len(combined_audio) / (sample_rate * 2)
            logger.info(f"Audio saved: {filename} ({duration:.2f}s, {len(combined_audio)} bytes)")
            
        except Exception as e:
            logger.error(f"Save error: {e}")
    
    async def start_server(self):
        """Start the WebSocket server"""
        hostname = get_server_hostname()
        
        logger.info("WebSocket Test Receiver Starting...")
        logger.info(f"Server: ws://{hostname}:{WEBSOCKET_PORT}")
        logger.info(f"Token: {self.EXPECTED_SESSION_TOKEN}")
        logger.info(f"Audio output: timestamped WAV files")
        logger.info("Press Ctrl+C to stop")
        
        async with websockets.serve(self.handle_client, "0.0.0.0", WEBSOCKET_PORT):
            logger.info("Server ready, waiting for connections...")
            await asyncio.Future()

async def main():
    receiver = WebSocketTestReceiver()
    await receiver.start_server()

if __name__ == "__main__":
    try:
        asyncio.run(main())
    except KeyboardInterrupt:
        logger.info("Server stopped")
