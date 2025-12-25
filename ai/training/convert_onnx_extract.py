"""
Transfer weights from your fine-tuned PyTorch model to an existing ONNX model.
This uses the pre-converted ONNX structure from onnx-community.
"""

import onnx
import torch
import numpy as np
from transformers import AutoModelForCausalLM
from pathlib import Path
import argparse

def load_pytorch_model(model_path):
    """Load the fine-tuned PyTorch model"""
    print(f"📥 Loading PyTorch model from {model_path}")
    model = AutoModelForCausalLM.from_pretrained(
        model_path,
        torch_dtype=torch.float32,
        device_map="cpu",
        trust_remote_code=True
    )
    return model

def get_pytorch_weights(model):
    """Extract weights from PyTorch model as numpy arrays"""
    print("🔄 Extracting weights from PyTorch model...")
    weights = {}
    for name, param in model.named_parameters():
        weights[name] = param.detach().cpu().numpy()
    print(f"✓ Extracted {len(weights)} weight tensors")
    return weights

def update_onnx_weights(onnx_path, pytorch_weights, output_path):
    """Update ONNX model with PyTorch weights"""
    print(f"📥 Loading ONNX model from {onnx_path}")
    onnx_model = onnx.load(onnx_path)
    
    print("🔄 Updating ONNX model weights...")
    
    # Create a mapping of ONNX initializer names to PyTorch parameter names
    # This may need adjustment based on naming conventions
    updated_count = 0
    
    for initializer in onnx_model.graph.initializer:
        # Try to find matching PyTorch weight
        # Common transformations: dots to underscores, etc.
        onnx_name = initializer.name
        
        # Try various name matching strategies
        pytorch_name = None
        for pt_name in pytorch_weights.keys():
            # Direct match
            if pt_name in onnx_name or onnx_name in pt_name:
                pytorch_name = pt_name
                break
            # Try with model. prefix
            if f"model.{pt_name}" == onnx_name:
                pytorch_name = pt_name
                break
        
        if pytorch_name:
            pytorch_weight = pytorch_weights[pytorch_name]
            
            # Check shape compatibility
            onnx_shape = tuple(initializer.dims)
            pytorch_shape = pytorch_weight.shape
            
            if onnx_shape == pytorch_shape:
                # Update the initializer with new weights
                initializer.raw_data = pytorch_weight.tobytes()
                updated_count += 1
                print(f"✓ Updated: {onnx_name} <- {pytorch_name} {pytorch_shape}")
            else:
                print(f"⚠️ Shape mismatch: {onnx_name} {onnx_shape} vs {pytorch_name} {pytorch_shape}")
        else:
            print(f"⚠️ No match found for ONNX weight: {onnx_name}")
    
    print(f"\n✅ Updated {updated_count} weights")
    
    # Save the updated model
    print(f"💾 Saving updated ONNX model to {output_path}")
    onnx.save(onnx_model, output_path)
    
    # Verify the model
    print("🧪 Verifying ONNX model...")
    onnx.checker.check_model(output_path)
    print("✅ Model verification passed!")

def main():
    parser = argparse.ArgumentParser(description="Transfer weights from PyTorch to ONNX model")
    parser.add_argument("--pytorch_model", type=str, required=True,
                        help="Path to fine-tuned PyTorch model")
    parser.add_argument("--onnx_model", type=str, required=True,
                        help="Path to base ONNX model (from onnx-community)")
    parser.add_argument("--output", type=str, default="./model_updated.onnx",
                        help="Output path for updated ONNX model")
    
    args = parser.parse_args()
    
    # Load PyTorch model and extract weights
    pytorch_model = load_pytorch_model(args.pytorch_model)
    pytorch_weights = get_pytorch_weights(pytorch_model)
    
    # Update ONNX model
    update_onnx_weights(args.onnx_model, pytorch_weights, args.output)
    
    print(f"\n🎉 Successfully created updated ONNX model at {args.output}")

if __name__ == "__main__":
    main()