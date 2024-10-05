import unittest
from modules.tts import generate_audio_file_path

class TestTTS(unittest.TestCase):
    def test_audio_file_path(self):
        context = "test mix"
        self.assertRegex(generate_audio_file_path(context), '^testmix')

if __name__ == '__main__':
    unittest.main()
