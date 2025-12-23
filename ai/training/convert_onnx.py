import os
import argparse
import sys
import subprocess
import shutil

# 1. Aggressive Dependency Check & Upgrade
def check_dependencies():
    print("Checking dependencies...")
    try:
        import optimum
        from optimum.exporters.onnx import main_export
        print(f"✓ Optimum version: {optimum.__version__}")
    except (ImportError, AttributeError):
        print("⚠️ Optimum/main_export not found or broken. Reinstalling...")
        subprocess.check_call([
            sys.executable, "-m", "pip", "install", 
            "--upgrade", "--force-reinstall",
            "optimum[onnxruntime]>=1.23.0", 
            "transformers>=4.47.0",
            "onnx>=1.16.0"
        ])
        print("✅ Dependencies reinstalled. Restarting script to apply changes...")
        # Restart the script to ensure the new imports are picked up correctly
        os.execv(sys.executable, ['python'] + sys.argv)

if __name__ == "__main__" or "SM_CHANNEL_MODEL" in os.environ:
    # Only run dependency check if we haven't already restarted
    if "DEPS_CHECKED" not in os.environ:
        os.environ["DEPS_CHECKED"] = "1"
        check_dependencies()

import torch
from transformers import AutoModelForCausalLM, AutoTokenizer
from peft import PeftModel
from optimum.exporters.onnx import main_export

def convert_to_onnx(model_path, output_path, task="text-generation-with-past"):
    """
    Convert a model to ONNX format using the programmatic API.
    """
    print(f"🚀 Converting model from {model_path} to ONNX format...")
    
    try:
        # Use the programmatic API instead of optimum-cli
        main_export(
            model_name_or_path=model_path,
            output=output_path,
            task=task,
            do_validation=False, # Speed up and avoid extra memory
            no_post_process=True
        )
        print(f"✅ Conversion successful! Output at: {output_path}")
    except Exception as e:
        print(f"❌ Error during conversion: {str(e)}")
        import traceback
        traceback.print_exc()
        raise RuntimeError(f"ONNX conversion failed: {e}")

def main():
    parser = argparse.ArgumentParser(description="Convert fine-tuned FunctionGemma to ONNX for WebGPU")
    parser.add_argument("--model_id", type=str, required=True, 
                        help="Base model ID (e.g., google/functiongemma-270m-it)")
    parser.add_argument("--finetuned_model_path", type=str, 
                        default=os.environ.get("SM_CHANNEL_MODEL", None), 
                        help="Path to the fine-tuned model weights")
    parser.add_argument("--output_dir", type=str, 
                        default=os.environ.get("SM_MODEL_DIR", "./onnx_output"), 
                        help="Directory to save the ONNX model")
    
    args = parser.parse_args()
    
    if not args.finetuned_model_path:
        raise ValueError("--finetuned_model_path is required")
    
    if not os.path.exists(args.finetuned_model_path):
        raise ValueError(f"Fine-tuned model path does not exist: {args.finetuned_model_path}")
    
    model_to_export_path = args.finetuned_model_path
    temp_merged_dir = "/tmp/merged_model"
    
    is_lora = os.path.exists(os.path.join(args.finetuned_model_path, "adapter_config.json"))
    
    if is_lora:
        print("ℹ️ LoRA adapter detected. Merging with base model...")
        tokenizer = AutoTokenizer.from_pretrained(args.model_id, trust_remote_code=True)
        base_model = AutoModelForCausalLM.from_pretrained(
            args.model_id,
            torch_dtype=torch.float32,
            device_map="cpu",
            trust_remote_code=True,
            low_cpu_mem_usage=True
        )
        model = PeftModel.from_pretrained(base_model, args.finetuned_model_path)
        merged_model = model.merge_and_unload()
        
        os.makedirs(temp_merged_dir, exist_ok=True)
        merged_model.save_pretrained(temp_merged_dir, safe_serialization=True)
        tokenizer.save_pretrained(temp_merged_dir)
        model_to_export_path = temp_merged_dir
        
        del base_model
        del model
        del merged_model
        torch.cuda.empty_cache() if torch.cuda.is_available() else None
        print(f"✅ Model merged and saved to {temp_merged_dir}")
    else:
        print("ℹ️ Full model detected.")
        tokenizer_config_path = os.path.join(args.finetuned_model_path, "tokenizer_config.json")
        if not os.path.exists(tokenizer_config_path):
            print("⚠️ Copying tokenizer from base model...")
            tokenizer = AutoTokenizer.from_pretrained(args.model_id, trust_remote_code=True)
            tokenizer.save_pretrained(args.finetuned_model_path)

    os.makedirs(args.output_dir, exist_ok=True)
    convert_to_onnx(model_to_export_path, args.output_dir)
    
    if os.path.exists(temp_merged_dir) and temp_merged_dir != args.finetuned_model_path:
        print(f"🧹 Cleaning up: {temp_merged_dir}")
        shutil.rmtree(temp_merged_dir)
        
    print(f"🎉 Process completed. ONNX model is in {args.output_dir}")

if __name__ == "__main__":
    main()
