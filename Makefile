ifneq (,$(wildcard ./.env))
    include .env
    export
endif

init:
	@echo "Initializing the environment and installing dependencies..."
	python3 -m venv venv
	source venv/bin/activate && python -m pip install -r requirements.txt

test:
	@echo "Running tests..."
	pytest -s

run:
	@echo "Running Xarvis..."
	python main.py

.PHONY: init test run
