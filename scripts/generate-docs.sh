#!/bin/bash

# Generate Swagger documentation
echo "Generating Swagger documentation..."
swag init -g cmd/api/main.go -o docs --parseInternal

echo "Swagger documentation generated successfully!"
echo "API docs will be available at: http://localhost:8088/swagger/index.html"
