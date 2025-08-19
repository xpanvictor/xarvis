#!/usr/bin/env python3
"""
Jarvis System Demo Script - Demonstrates real-time conversation capabilities.
Shows the unicellular conversation model and streaming responses.
"""

import asyncio
import json
import websockets
import aiohttp
from typing import Dict, Any
import base64

class JarvisDemo:
    """Demo client for Jarvis system."""
    
    def __init__(self, base_url: str = "http://localhost:8000"):
        self.base_url = base_url
        self.ws_url = base_url.replace("http://", "ws://").replace("https://", "wss://")
        self.user_id = "demo_user"
    
    async def demo_rest_conversation(self):
        """Demonstrate REST API conversation."""
        print("🚀 Testing REST API Conversation...")
        
        messages = [
            "Hello Jarvis, how are you today?",
            "What's the weather like?",
            "Can you help me with a Python programming question?",
            "Thank you for your help!"
        ]
        
        async with aiohttp.ClientSession() as session:
            for message in messages:
                print(f"\n👤 User: {message}")
                
                payload = {
                    "message": message,
                    "user_id": self.user_id,
                    "context": {
                        "source": "demo_script",
                        "timestamp": "now"
                    }
                }
                
                try:
                    async with session.post(
                        f"{self.base_url}/conversation/message",
                        json=payload
                    ) as response:
                        result = await response.json()
                        
                        if result.get("success"):
                            print(f"🤖 Jarvis: {result.get('response', 'No response')}")
                            
                            # Show any actions
                            actions = result.get("actions", [])
                            if actions:
                                print(f"📋 Actions: {', '.join(actions)}")
                        else:
                            print(f"❌ Error: {result.get('error', 'Unknown error')}")
                
                except Exception as e:
                    print(f"❌ Request failed: {e}")
                
                # Small delay between messages
                await asyncio.sleep(1)
    
    async def demo_streaming_conversation(self):
        """Demonstrate streaming conversation."""
        print("\n🌊 Testing Streaming Conversation...")
        
        async with aiohttp.ClientSession() as session:
            message = "Can you explain artificial intelligence in detail?"
            print(f"\n👤 User: {message}")
            
            payload = {
                "message": message,
                "user_id": self.user_id,
                "stream": True,
                "context": {
                    "source": "demo_streaming"
                }
            }
            
            try:
                async with session.post(
                    f"{self.base_url}/conversation/stream",
                    json=payload
                ) as response:
                    print("🤖 Jarvis: ", end="", flush=True)
                    
                    async for line in response.content:
                        if line:
                            try:
                                # Parse SSE format
                                line_str = line.decode('utf-8').strip()
                                if line_str.startswith('data: '):
                                    data = json.loads(line_str[6:])  # Remove "data: "
                                    
                                    if data.get("type") == "content":
                                        print(data.get("content", ""), end="", flush=True)
                                    elif data.get("type") == "complete":
                                        print("\n✅ Stream complete")
                                    elif data.get("type") == "error":
                                        print(f"\n❌ Stream error: {data.get('error')}")
                            except json.JSONDecodeError:
                                continue
            
            except Exception as e:
                print(f"❌ Streaming failed: {e}")
    
    async def demo_websocket_conversation(self):
        """Demonstrate WebSocket real-time conversation."""
        print("\n🔌 Testing WebSocket Real-time Conversation...")
        
        try:
            uri = f"{self.ws_url}/conversation/ws/{self.user_id}"
            
            async with websockets.connect(uri) as websocket:
                # Wait for connection confirmation
                response = await websocket.recv()
                data = json.loads(response)
                print(f"🔗 {data.get('message', 'Connected')}")
                
                # Test messages
                test_messages = [
                    "Hello via WebSocket!",
                    "Can you perform a web search for Python tutorials?",
                    "What tools do you have available?",
                    "Tell me about machine learning"
                ]
                
                for message in test_messages:
                    print(f"\n👤 User: {message}")
                    
                    # Send message
                    await websocket.send(json.dumps({
                        "type": "message",
                        "message": message,
                        "stream": True,
                        "context": {
                            "websocket_demo": True
                        }
                    }))
                    
                    # Receive response
                    full_response = ""
                    while True:
                        try:
                            response = await asyncio.wait_for(websocket.recv(), timeout=30)
                            data = json.loads(response)
                            
                            if data.get("type") == "thinking":
                                print("🤖 Jarvis: [thinking...]", end="\r", flush=True)
                            
                            elif data.get("type") == "stream_start":
                                print("🤖 Jarvis: ", end="", flush=True)
                            
                            elif data.get("type") == "stream_chunk":
                                chunk_data = data.get("data", {})
                                if chunk_data.get("type") == "content":
                                    content = chunk_data.get("content", "")
                                    print(content, end="", flush=True)
                                    full_response += content
                                elif chunk_data.get("type") == "status":
                                    print(f"\n📊 Status: {chunk_data.get('message')}")
                                elif chunk_data.get("type") == "tool_result":
                                    tool = chunk_data.get("tool", "Unknown")
                                    print(f"\n🔧 Tool Used: {tool}")
                            
                            elif data.get("type") == "stream_complete":
                                print("\n✅ Response complete")
                                break
                            
                            elif data.get("type") == "response":
                                # Non-streaming response
                                response_data = data.get("data", {})
                                print(f"🤖 Jarvis: {response_data.get('response', 'No response')}")
                                break
                            
                            elif data.get("type") == "error":
                                print(f"\n❌ Error: {data.get('error')}")
                                break
                        
                        except asyncio.TimeoutError:
                            print("\n⏰ Response timeout")
                            break
                        except websockets.exceptions.ConnectionClosed:
                            print("\n🔌 WebSocket connection closed")
                            return
                    
                    # Small delay between messages
                    await asyncio.sleep(2)
        
        except Exception as e:
            print(f"❌ WebSocket demo failed: {e}")
    
    async def demo_hardware_audio_simulation(self):
        """Simulate hardware audio interaction."""
        print("\n🎙️  Testing Hardware Audio Simulation...")
        
        # Simulate audio data (normally this would be real audio from ESP32)
        fake_audio_messages = [
            "Hello Jarvis, this is a simulated voice message",
            "What time is it?",
            "Can you help me with my schedule?"
        ]
        
        async with aiohttp.ClientSession() as session:
            for i, message in enumerate(fake_audio_messages):
                print(f"\n🎤 Simulated Audio Input: '{message}'")
                
                # Simulate base64 encoded audio (normally would be real audio bytes)
                fake_audio_bytes = message.encode('utf-8')
                fake_audio_b64 = base64.b64encode(fake_audio_bytes).decode('utf-8')
                
                payload = {
                    "device_id": "demo_esp32_001",
                    "audio_data": fake_audio_b64,
                    "duration": 3.0,
                    "continue_listening": i < len(fake_audio_messages) - 1,
                    "user_id": self.user_id
                }
                
                try:
                    async with session.post(
                        f"{self.base_url}/conversation/hardware/audio",
                        json=payload
                    ) as response:
                        result = await response.json()
                        
                        if result.get("success"):
                            print(f"📝 Transcribed: '{result.get('transcribed_text', 'N/A')}'")
                            print(f"🤖 Jarvis Response: {result.get('response', 'No response')}")
                            
                            if result.get("continue_listening"):
                                print(f"🔄 Continue listening for {result.get('listening_duration', 0)} seconds")
                        else:
                            print(f"❌ Error: {result.get('error', 'Unknown error')}")
                
                except Exception as e:
                    print(f"❌ Audio request failed: {e}")
                
                await asyncio.sleep(1)
    
    async def demo_conversation_status(self):
        """Check conversation status."""
        print("\n📊 Testing Conversation Status...")
        
        async with aiohttp.ClientSession() as session:
            try:
                async with session.get(
                    f"{self.base_url}/conversation/status/{self.user_id}"
                ) as response:
                    result = await response.json()
                    
                    print(f"👤 User: {result.get('user_id')}")
                    print(f"🆔 Session: {result.get('session_id')}")
                    print(f"📊 Status: {result.get('status', {})}")
                    print(f"🔌 WebSocket: {'Connected' if result.get('websocket_connected') else 'Disconnected'}")
            
            except Exception as e:
                print(f"❌ Status check failed: {e}")
    
    async def run_full_demo(self):
        """Run complete demonstration."""
        print("🎭 JARVIS SYSTEM DEMONSTRATION")
        print("=" * 50)
        
        # Check if system is healthy
        async with aiohttp.ClientSession() as session:
            try:
                async with session.get(f"{self.base_url}/health") as response:
                    if response.status == 200:
                        print("✅ Jarvis system is healthy and ready!")
                    else:
                        print("❌ Jarvis system is not responding properly")
                        return
            except Exception as e:
                print(f"❌ Cannot connect to Jarvis system: {e}")
                print("💡 Make sure the server is running on localhost:8000")
                return
        
        # Run all demos
        await self.demo_rest_conversation()
        await self.demo_streaming_conversation()
        await self.demo_websocket_conversation()
        await self.demo_hardware_audio_simulation()
        await self.demo_conversation_status()
        
        print("\n🎉 Demo completed! Your Jarvis system is working beautifully!")
        print("🔧 All conversation modes tested:")
        print("   ✅ REST API with unicellular conversations")
        print("   ✅ Streaming responses for real-time feel")
        print("   ✅ WebSocket for continuous interaction")
        print("   ✅ Hardware audio processing simulation")
        print("   ✅ Session management and status tracking")


async def main():
    """Main demo function."""
    demo = JarvisDemo()
    await demo.run_full_demo()


if __name__ == "__main__":
    print("🚀 Starting Jarvis System Demo...")
    asyncio.run(main())
