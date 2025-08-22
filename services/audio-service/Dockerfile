FROM python:3.12-slim

WORKDIR /app

# Install dependencies
COPY requirements.txt ./
RUN pip install --no-cache-dir -r requirements.txt

# Copy STT/TTS scripts
COPY . .

# Expose port for audio API
EXPOSE 5005

CMD ["python", "audio_server.py"]