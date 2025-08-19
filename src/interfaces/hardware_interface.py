"""
Hardware Interface - ESP32 and device communication management.
Handles serial communication, MQTT messaging, and hardware control.
"""

from typing import Dict, Any, List, Optional, Union, Callable
from datetime import datetime, timedelta
from dataclasses import dataclass
from enum import Enum
import asyncio
import json
import uuid
import threading
from concurrent.futures import ThreadPoolExecutor
from pathlib import Path
import time

from src.utils.logging import get_logger, with_correlation_id
from src.config import settings


logger = get_logger(__name__)


class HardwareType(Enum):
    """Types of hardware devices."""
    ESP32 = "esp32"
    ARDUINO = "arduino"
    RASPBERRY_PI = "raspberry_pi"
    SENSOR = "sensor"
    ACTUATOR = "actuator"
    CAMERA = "camera"
    MICROPHONE = "microphone"
    SPEAKER = "speaker"


class ConnectionType(Enum):
    """Hardware connection types."""
    SERIAL = "serial"
    MQTT = "mqtt"
    HTTP = "http"
    WEBSOCKET = "websocket"
    BLUETOOTH = "bluetooth"
    WIFI = "wifi"


class DeviceStatus(Enum):
    """Device status."""
    CONNECTED = "connected"
    DISCONNECTED = "disconnected"
    ERROR = "error"
    BUSY = "busy"
    IDLE = "idle"
    UNKNOWN = "unknown"


@dataclass
class HardwareDevice:
    """Represents a hardware device."""
    id: str
    name: str
    type: HardwareType
    connection_type: ConnectionType
    address: str
    status: DeviceStatus
    capabilities: List[str]
    last_seen: datetime
    metadata: Dict[str, Any]


@dataclass
class HardwareMessage:
    """Message to/from hardware device."""
    device_id: str
    message_type: str
    payload: Dict[str, Any]
    timestamp: datetime
    correlation_id: str
    priority: int = 0


class HardwareInterface:
    """
    Hardware interface for ESP32 and other device communications.
    Manages serial connections, MQTT messaging, and device control.
    """
    
    def __init__(self):
        self.devices: Dict[str, HardwareDevice] = {}
        self.serial_connections: Dict[str, Any] = {}
        self.mqtt_client = None
        self.is_initialized = False
        self.executor = ThreadPoolExecutor(max_workers=4)
        self.message_handlers: Dict[str, Callable] = {}
        self.message_queue = asyncio.Queue()
        self.outbound_queue = asyncio.Queue()
        
        # Connection settings
        self.esp32_port = settings.hardware.esp32_serial_port
        self.esp32_baud = settings.hardware.esp32_baud_rate
        self.mqtt_broker = settings.hardware.mqtt_broker
        self.mqtt_port = settings.hardware.mqtt_port
        
        # Data directory
        self.hardware_data_path = Path("data/hardware")
    
    async def initialize(self) -> None:
        """Initialize hardware interface."""
        logger.info("Initializing Hardware Interface")
        
        try:
            # Create data directory
            self.hardware_data_path.mkdir(parents=True, exist_ok=True)
            
            # Initialize serial connections
            await self._initialize_serial()
            
            # Initialize MQTT
            await self._initialize_mqtt()
            
            # Register message handlers
            await self._register_message_handlers()
            
            # Start communication workers
            asyncio.create_task(self._message_processor())
            asyncio.create_task(self._outbound_processor())
            asyncio.create_task(self._device_monitor())
            
            # Load device configurations
            await self._load_device_configurations()
            
            # Start device discovery
            asyncio.create_task(self._device_discovery())
            
            self.is_initialized = True
            logger.info("Hardware Interface initialized successfully")
            
        except Exception as e:
            logger.error("Failed to initialize Hardware Interface", error=str(e))
            raise
    
    async def send_command(
        self,
        device_id: str,
        command: str,
        parameters: Optional[Dict[str, Any]] = None,
        timeout: float = 5.0
    ) -> Dict[str, Any]:
        """
        Send command to hardware device.
        
        Args:
            device_id: Target device ID
            command: Command to send
            parameters: Command parameters
            timeout: Response timeout in seconds
        
        Returns:
            Command response
        """
        correlation_id = str(uuid.uuid4())
        
        with with_correlation_id(correlation_id):
            logger.info("Sending hardware command", 
                       device_id=device_id,
                       command=command,
                       parameters=parameters)
            
            try:
                if device_id not in self.devices:
                    return {
                        "success": False,
                        "error": f"Device {device_id} not found"
                    }
                
                device = self.devices[device_id]
                
                if device.status not in [DeviceStatus.CONNECTED, DeviceStatus.IDLE]:
                    return {
                        "success": False,
                        "error": f"Device {device_id} not available (status: {device.status.value})"
                    }
                
                # Create message
                message = HardwareMessage(
                    device_id=device_id,
                    message_type="command",
                    payload={
                        "command": command,
                        "parameters": parameters or {},
                        "correlation_id": correlation_id
                    },
                    timestamp=datetime.now(),
                    correlation_id=correlation_id
                )
                
                # Send message
                await self.outbound_queue.put(message)
                
                # Wait for response (simplified - in practice you'd have a proper response handler)
                response = await self._wait_for_response(correlation_id, timeout)
                
                logger.info("Hardware command completed", 
                           device_id=device_id,
                           success=response.get("success", False))
                
                return response
                
            except Exception as e:
                logger.error("Hardware command failed", 
                           device_id=device_id,
                           command=command,
                           error=str(e))
                
                return {
                    "success": False,
                    "error": str(e)
                }
    
    async def send_audio_request(
        self,
        device_id: str = "esp32_main",
        duration: float = 5.0
    ) -> Dict[str, Any]:
        """
        Request audio recording from hardware.
        
        Args:
            device_id: Target device ID
            duration: Recording duration in seconds
        
        Returns:
            Audio request response
        """
        return await self.send_command(
            device_id=device_id,
            command="record_audio",
            parameters={
                "duration": duration,
                "format": "wav",
                "sample_rate": settings.audio.sample_rate
            }
        )
    
    async def send_audio_playback(
        self,
        device_id: str = "esp32_main",
        audio_data: bytes = None,
        audio_url: str = None
    ) -> Dict[str, Any]:
        """
        Send audio for playback on hardware.
        
        Args:
            device_id: Target device ID
            audio_data: Audio data bytes
            audio_url: URL to audio file
        
        Returns:
            Playback response
        """
        if not audio_data and not audio_url:
            return {
                "success": False,
                "error": "No audio data or URL provided"
            }
        
        parameters = {}
        
        if audio_data:
            # Save audio temporarily and provide local URL
            audio_file = await self._save_temp_audio(audio_data)
            parameters["audio_url"] = f"/tmp/{audio_file.name}"
        else:
            parameters["audio_url"] = audio_url
        
        return await self.send_command(
            device_id=device_id,
            command="play_audio",
            parameters=parameters
        )
    
    async def get_device_status(self, device_id: str) -> Dict[str, Any]:
        """
        Get status of a hardware device.
        
        Args:
            device_id: Device ID
        
        Returns:
            Device status information
        """
        try:
            if device_id not in self.devices:
                return {
                    "success": False,
                    "error": f"Device {device_id} not found"
                }
            
            device = self.devices[device_id]
            
            return {
                "success": True,
                "device": {
                    "id": device.id,
                    "name": device.name,
                    "type": device.type.value,
                    "connection_type": device.connection_type.value,
                    "status": device.status.value,
                    "last_seen": device.last_seen.isoformat(),
                    "capabilities": device.capabilities,
                    "metadata": device.metadata
                }
            }
            
        except Exception as e:
            logger.error("Failed to get device status", device_id=device_id, error=str(e))
            return {
                "success": False,
                "error": str(e)
            }
    
    async def list_devices(
        self,
        device_type: Optional[HardwareType] = None,
        status: Optional[DeviceStatus] = None
    ) -> List[Dict[str, Any]]:
        """
        List available hardware devices.
        
        Args:
            device_type: Filter by device type
            status: Filter by status
        
        Returns:
            List of devices
        """
        devices = []
        
        for device in self.devices.values():
            # Apply filters
            if device_type and device.type != device_type:
                continue
            
            if status and device.status != status:
                continue
            
            devices.append({
                "id": device.id,
                "name": device.name,
                "type": device.type.value,
                "connection_type": device.connection_type.value,
                "status": device.status.value,
                "last_seen": device.last_seen.isoformat(),
                "capabilities": device.capabilities
            })
        
        return devices
    
    async def register_device(
        self,
        device_id: str,
        name: str,
        device_type: HardwareType,
        connection_type: ConnectionType,
        address: str,
        capabilities: List[str]
    ) -> bool:
        """
        Register a new hardware device.
        
        Args:
            device_id: Unique device ID
            name: Device name
            device_type: Type of device
            connection_type: Connection type
            address: Device address
            capabilities: Device capabilities
        
        Returns:
            True if successful
        """
        try:
            device = HardwareDevice(
                id=device_id,
                name=name,
                type=device_type,
                connection_type=connection_type,
                address=address,
                status=DeviceStatus.DISCONNECTED,
                capabilities=capabilities,
                last_seen=datetime.now(),
                metadata={}
            )
            
            self.devices[device_id] = device
            
            # Try to establish connection
            await self._connect_device(device)
            
            logger.info("Device registered", 
                       device_id=device_id,
                       device_type=device_type.value,
                       connection_type=connection_type.value)
            
            return True
            
        except Exception as e:
            logger.error("Device registration failed", device_id=device_id, error=str(e))
            return False
    
    async def unregister_device(self, device_id: str) -> bool:
        """
        Unregister a hardware device.
        
        Args:
            device_id: Device ID to unregister
        
        Returns:
            True if successful
        """
        try:
            if device_id not in self.devices:
                return False
            
            # Disconnect device
            await self._disconnect_device(device_id)
            
            # Remove from registry
            del self.devices[device_id]
            
            logger.info("Device unregistered", device_id=device_id)
            return True
            
        except Exception as e:
            logger.error("Device unregistration failed", device_id=device_id, error=str(e))
            return False
    
    async def cleanup(self) -> None:
        """Cleanup hardware interface resources."""
        logger.info("Cleaning up Hardware Interface")
        
        try:
            # Disconnect all devices
            for device_id in list(self.devices.keys()):
                await self._disconnect_device(device_id)
            
            # Close serial connections
            for connection in self.serial_connections.values():
                try:
                    if hasattr(connection, 'close'):
                        connection.close()
                except Exception as e:
                    logger.error("Failed to close serial connection", error=str(e))
            
            # Disconnect MQTT
            if self.mqtt_client:
                try:
                    await self._disconnect_mqtt()
                except Exception as e:
                    logger.error("Failed to disconnect MQTT", error=str(e))
            
            # Shutdown executor
            self.executor.shutdown(wait=True)
            
            logger.info("Hardware Interface cleanup completed")
            
        except Exception as e:
            logger.error("Hardware Interface cleanup failed", error=str(e))
    
    async def _initialize_serial(self) -> None:
        """Initialize serial connections."""
        try:
            # Try to connect to ESP32
            if await self._test_serial_port(self.esp32_port):
                import serial
                
                connection = serial.Serial(
                    port=self.esp32_port,
                    baudrate=self.esp32_baud,
                    timeout=1.0
                )
                
                self.serial_connections["esp32_main"] = connection
                
                # Register ESP32 device
                await self.register_device(
                    device_id="esp32_main",
                    name="Main ESP32 Device",
                    device_type=HardwareType.ESP32,
                    connection_type=ConnectionType.SERIAL,
                    address=self.esp32_port,
                    capabilities=["audio_recording", "audio_playback", "sensors"]
                )
                
                logger.info("ESP32 serial connection established", port=self.esp32_port)
            
            else:
                logger.warning("ESP32 not found on configured port", port=self.esp32_port)
                
        except ImportError:
            logger.warning("pyserial not available, serial connections disabled")
        except Exception as e:
            logger.error("Serial initialization failed", error=str(e))
    
    async def _test_serial_port(self, port: str) -> bool:
        """Test if serial port is available."""
        try:
            import serial
            
            with serial.Serial(port, self.esp32_baud, timeout=1.0) as test_conn:
                test_conn.write(b"AT\n")
                response = test_conn.readline()
                return len(response) > 0
        
        except Exception:
            return False
    
    async def _initialize_mqtt(self) -> None:
        """Initialize MQTT client."""
        try:
            import paho.mqtt.client as mqtt
            
            self.mqtt_client = mqtt.Client()
            
            # Set callbacks
            self.mqtt_client.on_connect = self._on_mqtt_connect
            self.mqtt_client.on_message = self._on_mqtt_message
            self.mqtt_client.on_disconnect = self._on_mqtt_disconnect
            
            # Connect to broker
            self.mqtt_client.connect(self.mqtt_broker, self.mqtt_port, 60)
            self.mqtt_client.loop_start()
            
            logger.info("MQTT client initialized", broker=self.mqtt_broker)
            
        except ImportError:
            logger.warning("paho-mqtt not available, MQTT disabled")
            self.mqtt_client = None
        except Exception as e:
            logger.error("MQTT initialization failed", error=str(e))
            self.mqtt_client = None
    
    def _on_mqtt_connect(self, client, userdata, flags, rc):
        """MQTT connection callback."""
        if rc == 0:
            logger.info("MQTT connected successfully")
            # Subscribe to device topics
            client.subscribe("xarvis/devices/+/status")
            client.subscribe("xarvis/devices/+/audio")
            client.subscribe("xarvis/devices/+/response")
        else:
            logger.error("MQTT connection failed", return_code=rc)
    
    def _on_mqtt_message(self, client, userdata, msg):
        """MQTT message callback."""
        try:
            topic = msg.topic
            payload = json.loads(msg.payload.decode())
            
            # Parse topic to extract device ID
            topic_parts = topic.split('/')
            if len(topic_parts) >= 3:
                device_id = topic_parts[2]
                message_type = topic_parts[3] if len(topic_parts) > 3 else "unknown"
                
                # Create hardware message
                message = HardwareMessage(
                    device_id=device_id,
                    message_type=message_type,
                    payload=payload,
                    timestamp=datetime.now(),
                    correlation_id=payload.get("correlation_id", str(uuid.uuid4()))
                )
                
                # Queue for processing
                asyncio.create_task(self.message_queue.put(message))
        
        except Exception as e:
            logger.error("MQTT message processing failed", error=str(e))
    
    def _on_mqtt_disconnect(self, client, userdata, rc):
        """MQTT disconnect callback."""
        logger.warning("MQTT disconnected", return_code=rc)
    
    async def _disconnect_mqtt(self) -> None:
        """Disconnect MQTT client."""
        if self.mqtt_client:
            self.mqtt_client.loop_stop()
            self.mqtt_client.disconnect()
    
    async def _register_message_handlers(self) -> None:
        """Register message handlers."""
        self.message_handlers = {
            "status": self._handle_status_message,
            "audio": self._handle_audio_message,
            "response": self._handle_response_message,
            "sensor": self._handle_sensor_message
        }
    
    async def _handle_status_message(self, message: HardwareMessage) -> None:
        """Handle status message from device."""
        device_id = message.device_id
        
        if device_id in self.devices:
            device = self.devices[device_id]
            
            # Update device status
            status_str = message.payload.get("status", "unknown")
            try:
                device.status = DeviceStatus(status_str)
            except ValueError:
                device.status = DeviceStatus.UNKNOWN
            
            device.last_seen = datetime.now()
            device.metadata.update(message.payload.get("metadata", {}))
            
            logger.debug("Device status updated", 
                        device_id=device_id,
                        status=device.status.value)
    
    async def _handle_audio_message(self, message: HardwareMessage) -> None:
        """Handle audio message from device."""
        logger.info("Received audio message from device", device_id=message.device_id)
        
        # Process audio data
        audio_data = message.payload.get("audio_data")
        if audio_data:
            # This would typically be forwarded to the audio interface
            logger.info("Audio data received", 
                       device_id=message.device_id,
                       data_size=len(audio_data))
    
    async def _handle_response_message(self, message: HardwareMessage) -> None:
        """Handle response message from device."""
        correlation_id = message.correlation_id
        logger.debug("Received response from device", 
                    device_id=message.device_id,
                    correlation_id=correlation_id)
        
        # Store response for correlation (simplified implementation)
        # In practice, you'd have a proper response correlation system
    
    async def _handle_sensor_message(self, message: HardwareMessage) -> None:
        """Handle sensor data message."""
        logger.debug("Received sensor data", 
                    device_id=message.device_id,
                    sensors=list(message.payload.keys()))
        
        # Process sensor data
        sensor_data = message.payload
        
        # This could trigger actions based on sensor readings
        # For example, motion detection, temperature alerts, etc.
    
    async def _message_processor(self) -> None:
        """Process incoming hardware messages."""
        while True:
            try:
                message = await self.message_queue.get()
                
                # Find appropriate handler
                handler = self.message_handlers.get(message.message_type)
                if handler:
                    await handler(message)
                else:
                    logger.warning("No handler for message type", 
                                 message_type=message.message_type)
                
                self.message_queue.task_done()
                
            except Exception as e:
                logger.error("Message processing failed", error=str(e))
    
    async def _outbound_processor(self) -> None:
        """Process outbound messages to hardware."""
        while True:
            try:
                message = await self.outbound_queue.get()
                
                # Send message via appropriate connection
                await self._send_message(message)
                
                self.outbound_queue.task_done()
                
            except Exception as e:
                logger.error("Outbound message processing failed", error=str(e))
    
    async def _send_message(self, message: HardwareMessage) -> None:
        """Send message to hardware device."""
        device_id = message.device_id
        
        if device_id not in self.devices:
            logger.error("Device not found", device_id=device_id)
            return
        
        device = self.devices[device_id]
        
        try:
            if device.connection_type == ConnectionType.SERIAL:
                await self._send_serial_message(device, message)
            elif device.connection_type == ConnectionType.MQTT:
                await self._send_mqtt_message(device, message)
            else:
                logger.error("Unsupported connection type", 
                           connection_type=device.connection_type.value)
        
        except Exception as e:
            logger.error("Failed to send message", 
                        device_id=device_id, 
                        error=str(e))
    
    async def _send_serial_message(self, device: HardwareDevice, message: HardwareMessage) -> None:
        """Send message via serial connection."""
        connection = self.serial_connections.get(device.id)
        
        if not connection:
            logger.error("Serial connection not found", device_id=device.id)
            return
        
        try:
            # Format message as JSON
            message_data = {
                "type": message.message_type,
                "payload": message.payload,
                "timestamp": message.timestamp.isoformat(),
                "correlation_id": message.correlation_id
            }
            
            message_json = json.dumps(message_data)
            message_bytes = (message_json + '\n').encode('utf-8')
            
            # Send via serial
            loop = asyncio.get_event_loop()
            await loop.run_in_executor(
                self.executor,
                connection.write,
                message_bytes
            )
            
            logger.debug("Serial message sent", device_id=device.id)
        
        except Exception as e:
            logger.error("Serial message send failed", device_id=device.id, error=str(e))
    
    async def _send_mqtt_message(self, device: HardwareDevice, message: HardwareMessage) -> None:
        """Send message via MQTT."""
        if not self.mqtt_client:
            logger.error("MQTT client not available")
            return
        
        try:
            topic = f"xarvis/devices/{device.id}/commands"
            
            message_data = {
                "type": message.message_type,
                "payload": message.payload,
                "timestamp": message.timestamp.isoformat(),
                "correlation_id": message.correlation_id
            }
            
            payload = json.dumps(message_data)
            
            self.mqtt_client.publish(topic, payload)
            
            logger.debug("MQTT message sent", device_id=device.id, topic=topic)
        
        except Exception as e:
            logger.error("MQTT message send failed", device_id=device.id, error=str(e))
    
    async def _device_monitor(self) -> None:
        """Monitor device health and connectivity."""
        while True:
            try:
                current_time = datetime.now()
                
                for device in self.devices.values():
                    # Check if device is stale
                    time_since_last_seen = current_time - device.last_seen
                    
                    if time_since_last_seen > timedelta(minutes=5):
                        if device.status != DeviceStatus.DISCONNECTED:
                            device.status = DeviceStatus.DISCONNECTED
                            logger.warning("Device marked as disconnected", 
                                         device_id=device.id)
                
                # Sleep for monitoring interval
                await asyncio.sleep(30)  # Check every 30 seconds
                
            except Exception as e:
                logger.error("Device monitoring failed", error=str(e))
                await asyncio.sleep(60)  # Wait longer on error
    
    async def _device_discovery(self) -> None:
        """Discover new hardware devices."""
        while True:
            try:
                # This is where you'd implement device discovery
                # For example, scanning serial ports, MQTT discovery, etc.
                
                await asyncio.sleep(300)  # Run discovery every 5 minutes
                
            except Exception as e:
                logger.error("Device discovery failed", error=str(e))
                await asyncio.sleep(300)
    
    async def _connect_device(self, device: HardwareDevice) -> bool:
        """Connect to a hardware device."""
        try:
            if device.connection_type == ConnectionType.SERIAL:
                # Serial connection already handled in initialization
                device.status = DeviceStatus.CONNECTED
                return True
            
            elif device.connection_type == ConnectionType.MQTT:
                # MQTT devices connect automatically when they publish
                device.status = DeviceStatus.IDLE
                return True
            
            return False
            
        except Exception as e:
            logger.error("Device connection failed", 
                        device_id=device.id, 
                        error=str(e))
            device.status = DeviceStatus.ERROR
            return False
    
    async def _disconnect_device(self, device_id: str) -> None:
        """Disconnect from a hardware device."""
        try:
            if device_id in self.devices:
                device = self.devices[device_id]
                
                if device.connection_type == ConnectionType.SERIAL:
                    connection = self.serial_connections.get(device_id)
                    if connection and hasattr(connection, 'close'):
                        connection.close()
                    
                    if device_id in self.serial_connections:
                        del self.serial_connections[device_id]
                
                device.status = DeviceStatus.DISCONNECTED
                
        except Exception as e:
            logger.error("Device disconnection failed", 
                        device_id=device_id, 
                        error=str(e))
    
    async def _load_device_configurations(self) -> None:
        """Load device configurations from disk."""
        try:
            config_file = self.hardware_data_path / "devices.json"
            
            if config_file.exists():
                with open(config_file, 'r') as f:
                    configs = json.load(f)
                
                for config in configs:
                    await self.register_device(
                        device_id=config["id"],
                        name=config["name"],
                        device_type=HardwareType(config["type"]),
                        connection_type=ConnectionType(config["connection_type"]),
                        address=config["address"],
                        capabilities=config["capabilities"]
                    )
                
                logger.info("Device configurations loaded", count=len(configs))
        
        except Exception as e:
            logger.error("Failed to load device configurations", error=str(e))
    
    async def _save_temp_audio(self, audio_data: bytes) -> Path:
        """Save audio data to temporary file."""
        temp_file = self.hardware_data_path / f"temp_audio_{uuid.uuid4().hex}.wav"
        
        with open(temp_file, 'wb') as f:
            f.write(audio_data)
        
        return temp_file
    
    async def _wait_for_response(
        self,
        correlation_id: str,
        timeout: float
    ) -> Dict[str, Any]:
        """Wait for response with correlation ID."""
        # Simplified implementation
        # In practice, you'd have a proper response correlation system
        
        await asyncio.sleep(min(timeout, 1.0))  # Simulate response time
        
        return {
            "success": True,
            "result": "Command completed",
            "correlation_id": correlation_id
        }
    
    def get_status(self) -> Dict[str, Any]:
        """Get hardware interface status."""
        return {
            "initialized": self.is_initialized,
            "devices": len(self.devices),
            "connected_devices": len([d for d in self.devices.values() if d.status == DeviceStatus.CONNECTED]),
            "serial_connections": len(self.serial_connections),
            "mqtt_connected": self.mqtt_client is not None,
            "message_queue_size": self.message_queue.qsize(),
            "outbound_queue_size": self.outbound_queue.qsize()
        }
