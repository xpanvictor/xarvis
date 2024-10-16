import unittest
from modules.nlp import communicate_with_nlp

class TestTTS(unittest.TestCase):
    def test_nlp_response(self):
        resp = communicate_with_nlp("Tell me something fun!")
        print(resp)
        self.assertTrue(isinstance(resp, bytes))

if __name__ == '__main__':
    unittest.main()
