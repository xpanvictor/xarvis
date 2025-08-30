PROJECT_NAME = xarvis
DOCKER_COMPOSE = docker compose

# Default target
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  make up          - Start core services (Go app + MQTT broker)"
	@echo "  make down        - Stop all services"
	@echo "  make restart     - Restart xarvis-core"
	@echo "  make logs        - Tail logs of xarvis-core"
	@echo "  make proxy       - Add Traefik reverse proxy (/v1/* routes)"
	@echo "  make voice       - Add Whisper STT + Piper TTS"
	@echo "  make ai          - Add local AI services (Ollama, Embeddings-TEI)"
	@echo "  make qdrant      - Add Qdrant vector backend"
	@echo "  make tidb        - Add local TiDB for dev (Serverless preferred in prod)"
	@echo "  make clean       - Stop & remove all containers, networks, volumes"

.PHONY: up down restart logs proxy voice ai qdrant tidb clean

up:
	$(DOCKER_COMPOSE) up -d --build mosquitto xarvis-core

down:
	$(DOCKER_COMPOSE) down

restart:
	$(DOCKER_COMPOSE) restart xarvis-core

logs:
	$(DOCKER_COMPOSE) logs -f xarvis-core

proxy:
	$(DOCKER_COMPOSE) --profile proxy up -d traefik

voice:
	$(DOCKER_COMPOSE) --profile voice up -d stt-whisper tts-piper

ai:
	$(DOCKER_COMPOSE) --profile ai-local up -d ollama embeddings-tei

qdrant:
	$(DOCKER_COMPOSE) --profile vector-qdrant up -d qdrant

tidb:
	$(DOCKER_COMPOSE) --profile tidb-local up -d tidb-local

clean:
	$(DOCKER_COMPOSE) down -v --remove-orphans
