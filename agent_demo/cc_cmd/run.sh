#!/bin/bash

# Ensure dependencies are installed
if [ ! -d "node_modules" ]; then
    echo "Installing dependencies..."
    npm install
fi

# Run the demo
echo "Running Claude Pty Demo..."
node demo.js
