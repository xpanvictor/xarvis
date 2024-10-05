
init:
	@echo "Initializing the environment and installing dependencies..."
	python3 -m venv venv
	source venv/bin/activate && pip install -r requirements.txt

test:
	@echo "Running tests..."
	py.test tests

run:
	@echo "Running Xarvis..."
	python main.py

.PHONY: init test run
