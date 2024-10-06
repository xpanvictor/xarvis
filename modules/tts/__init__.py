# TTS module
import re
from datetime import datetime


def generate_audio(text: str) -> str:
    """
    TTS utility functionalities
    Generates audio wav files and stores into
    `generated` dir.
    Returns relative generated file path.
    @:returns: name of generated wav file
    """
    # TODO: the audio generation
    return generate_audio_file_path(text)


def generate_audio_file_path(context: str) -> str:
    mod_context = re.sub('[^A-Za-z0-9]+', '', context)
    timestamp = datetime.now().strftime("%Y%m%d-%H%M%S")
    return "-".join([mod_context, timestamp])
