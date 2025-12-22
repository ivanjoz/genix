#!/bin/bash

# Script to install dependencies for running the SageMaker launcher locally

# 1. Create a virtual environment if it doesn't exist
if [ ! -d "venv" ]; then
    echo "Creating virtual environment..."
    python3 -m venv venv
fi

# 2. Activate virtual environment
echo "Activating virtual environment..."
source venv/bin/activate

# 3. Upgrade pip
echo "Upgrading pip..."
pip install --upgrade pip

# 4. Install requirements
if [ -f "requirements.launcher.txt" ]; then
    echo "Installing dependencies from requirements.launcher.txt..."
    pip install -r requirements.launcher.txt
else
    echo "requirements.launcher.txt not found. Installing basic dependencies..."
    pip install sagemaker boto3
fi

echo "-------------------------------------------------------"
echo "Done! Dependencies installed successfully."
echo "To activate the environment in your terminal, run:"
echo "source venv/bin/activate"
echo "-------------------------------------------------------"

