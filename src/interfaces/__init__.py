"""
Interfaces package initialization.
Contains audio and hardware interface components.
"""

from .audio_interface import AudioInterface
from .hardware_interface import HardwareInterface

__all__ = ["AudioInterface", "HardwareInterface"]
