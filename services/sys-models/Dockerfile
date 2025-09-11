FROM python:3.11-slim as base

# Set working directory
WORKDIR /app

# Install system dependencies including audio backends
RUN apt-get update && apt-get install -y \
    gcc \
    g++ \
    ffmpeg \
    libsndfile1 \
    libsndfile1-dev \
    sox \
    libsox-dev \
    libasound2-dev \
    portaudio19-dev \
    curl \
    && rm -rf /var/lib/apt/lists/*

# Copy requirements first for better Docker layer caching
COPY requirements.txt .

# Install Python dependencies
RUN pip install --no-cache-dir -r requirements.txt

# Pre-download the Silero VAD model to avoid download during runtime
RUN python -c "import torch; torch.hub.load('snakers4/silero-vad', 'silero_vad', force_reload=False)"

# Expose port
EXPOSE 8001

# Development target with live reload
FROM base as development

# Install development dependencies
RUN pip install --no-cache-dir watchdog

# Health check
HEALTHCHECK --interval=30s --timeout=30s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8001/health || exit 1

# Run with auto-reload for development
CMD ["python", "-m", "uvicorn", "models_server:app", "--host", "0.0.0.0", "--port", "8001", "--reload", "--log-level", "debug"]

# Production target
FROM base as production

# Copy application code
COPY models_server.py .

# Health check
HEALTHCHECK --interval=30s --timeout=30s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8001/health || exit 1

# Run the application
CMD ["python", "models_server.py"]
