ifneq (,$(wildcard ./.env))
    include .env
    export
endif

# Development Commands
.PHONY: init dev test lint format clean build run docker-build docker-up docker-down logs help

init: ## Initialize the development environment
	@echo "🚀 Initializing Xarvis development environment..."
	python3 -m venv venv
	./venv/bin/pip install --upgrade pip
	./venv/bin/pip install -r requirements.txt
	@echo "✅ Environment initialized. Activate with: source venv/bin/activate"

dev: ## Run development server with hot reload
	@echo "🔥 Starting Xarvis development server..."
	uvicorn src.main:app --host 0.0.0.0 --port 8000 --reload

test: ## Run tests with coverage
	@echo "🧪 Running tests..."
	pytest tests/ -v --cov=src --cov-report=html --cov-report=term-missing

lint: ## Run code linting
	@echo "🔍 Running linters..."
	mypy src/
	black --check src/
	isort --check-only src/

format: ## Format code with black and isort
	@echo "✨ Formatting code..."
	black src/ tests/
	isort src/ tests/

clean: ## Clean up generated files
	@echo "🧹 Cleaning up..."
	find . -type f -name "*.pyc" -delete
	find . -type d -name "__pycache__" -delete
	find . -type d -name "*.egg-info" -exec rm -rf {} +
	rm -rf build/
	rm -rf dist/
	rm -rf .coverage
	rm -rf htmlcov/
	rm -rf .mypy_cache/
	rm -rf .pytest_cache/

build: ## Build the application
	@echo "🏗️  Building Xarvis..."
	python -m pip install build
	python -m build

# Docker Commands
docker-build: ## Build Docker containers
	@echo "🐳 Building Docker containers..."
	docker-compose build

docker-up: ## Start all services with Docker
	@echo "🚀 Starting Xarvis with Docker..."
	docker-compose up -d
	@echo "✅ Services started:"
	@echo "   - Main API: http://localhost:8000"
	@echo "   - Flower (Celery monitoring): http://localhost:5555"
	@echo "   - Health check: http://localhost:8000/health"

docker-down: ## Stop all Docker services
	@echo "🛑 Stopping Docker services..."
	docker-compose down

docker-logs: ## View Docker logs
	docker-compose logs -f

docker-shell: ## Access main container shell
	docker-compose exec xarvis-main bash

# Database Commands
db-upgrade: ## Run database migrations
	@echo "📊 Running database migrations..."
	alembic upgrade head

db-downgrade: ## Rollback database migration
	@echo "📊 Rolling back database migration..."
	alembic downgrade -1

db-migrate: ## Create new migration
	@echo "📊 Creating new migration..."
	alembic revision --autogenerate -m "$(MESSAGE)"

# Production Commands
run: ## Run production server
	@echo "🚀 Starting Xarvis production server..."
	uvicorn src.main:app --host 0.0.0.0 --port 8000 --workers 4

celery-worker: ## Start Celery worker
	@echo "⚡ Starting Celery worker..."
	celery -A src.backbone.job_runner.celery_app worker --loglevel=info

celery-beat: ## Start Celery beat scheduler
	@echo "⏰ Starting Celery beat scheduler..."
	celery -A src.backbone.job_runner.celery_app beat --loglevel=info

celery-flower: ## Start Flower monitoring
	@echo "🌸 Starting Flower monitoring..."
	celery -A src.backbone.job_runner.celery_app flower

# Utility Commands
logs: ## View application logs
	tail -f logs/xarvis.log

health: ## Check system health
	@echo "🩺 Checking Xarvis health..."
	curl -f http://localhost:8000/health || echo "❌ Service not responding"

setup-pre-commit: ## Setup pre-commit hooks
	@echo "⚙️  Setting up pre-commit hooks..."
	pre-commit install

help: ## Show this help
	@echo "Xarvis - AI Assistant System"
	@echo ""
	@echo "Available commands:"
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z_-]+:.*##/ { printf "  %-20s %s\n", $$1, $$2 }' $(MAKEFILE_LIST)
