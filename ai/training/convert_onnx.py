import os
import argparse
import sys
import subprocess
import shutil

# 1. Strict Environment Repair
def repair_environment():
    print("🛠️ Repairing environment with strict version pinning...")
    try:
        # Adjusted to transformers==4.46.1 to satisfy optimum[onnxruntime] requirements
        subprocess.check_call([
            sys.executable, "-m", "pip", "install", 
            "--upgrade", "--force-reinstall",
            "numpy==1.26.4",
            "protobuf==3.20.3",
            "torch==2.3.0",
            "transformers==4.46.1",
            "optimum[onnxruntime]==1.23.3", 
            "onnx==1.17.0"
        ])
        print("✅ Environment repaired with stable versions. Restarting...")
        # Mark as repaired to avoid infinite loop
        os.environ["ENV_REPAIRED"] = "1"
        # Use sys.executable to ensure we restart with the correct python binary
        os.execv(sys.executable, [sys.executable] + sys.argv)
    except Exception as e:
        print(f"❌ Failed to repair environment: {e}")
        sys.exit(1)

# Only run repair if we haven't already
if "ENV_REPAIRED" not in os.environ:
    repair_environment()

# Now it should be safe to import
import torch
from transformers import AutoModelForCausalLM, AutoTokenizer
from peft import PeftModel
from optimum.exporters.onnx import main_export

def convert_to_onnx(model_path, output_path, task="text-generation-with-past"):
    print(f"🚀 Converting model from {model_path} to ONNX format...")
    try:
        main_export(
            model_name_or_path=model_path,
            output=output_path,
            task=task,
            do_validation=False,
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
                        help="Base model ID")
    parser.add_argument("--finetuned_model_path", type=str, 
                        default=os.environ.get("SM_CHANNEL_MODEL", None))
    parser.add_argument("--output_dir", type=str, 
                        default=os.environ.get("SM_MODEL_DIR", "./onnx_output"))
    
    args = parser.parse_args()
    
    if not args.finetuned_model_path:
        raise ValueError("--finetuned_model_path is required")
    
    model_to_export_path = args.finetuned_model_path
    temp_merged_dir = "/tmp/merged_model"
    
    is_lora = os.path.exists(os.path.join(args.finetuned_model_path, "adapter_config.json"))
    
    if is_lora:
        print("ℹ️ LoRA adapter detected. Merging...")
        tokenizer = AutoTokenizer.from_pretrained(args.model_id, trust_remote_code=True)
        base_model = AutoModelForCausalLM.from_pretrained(
            args.model_id,
            torch_dtype=torch.float32,
            device_map="cpu",
            trust_remote_code=True
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
    else:
        print("ℹ️ Full model detected.")
        if not os.path.exists(os.path.join(args.finetuned_model_path, "tokenizer_config.json")):
            tokenizer = AutoTokenizer.from_pretrained(args.model_id, trust_remote_code=True)
            tokenizer.save_pretrained(args.finetuned_model_path)

    os.makedirs(args.output_dir, exist_ok=True)
    convert_to_onnx(model_to_export_path, args.output_dir)
    
    if os.path.exists(temp_merged_dir) and temp_merged_dir != args.finetuned_model_path:
        shutil.rmtree(temp_merged_dir)
        
    print(f"🎉 Process completed. ONNX model is in {args.output_dir}")

if __name__ == "__main__":
    main()
