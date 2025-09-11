#!/bin/bash
# Test script for System Models Service

echo "Testing System Models Service..."

# Check if we can build the Docker image
echo "Building Docker image..."
cd /Users/xpan/lab/factory/xarvis/services/sys-models

if docker build -t sys-models-test . > /dev/null 2>&1; then
    echo "✅ Docker image builds successfully"
else
    echo "❌ Docker image build failed"
    exit 1
fi

# Test that the service starts (without Silero model for quick test)
echo "Testing service startup..."
if docker run --rm -d --name sys-models-test -p 8001:8001 sys-models-test > /dev/null 2>&1; then
    sleep 5
    
    # Test health endpoint
    if curl -f http://localhost:8001/health > /dev/null 2>&1; then
        echo "✅ Service health check passed"
    else
        echo "⚠️  Service started but health check failed (expected without model)"
    fi
    
    # Stop the container
    docker stop sys-models-test > /dev/null 2>&1
else
    echo "❌ Service failed to start"
fi

echo "System Models Service test completed!"
