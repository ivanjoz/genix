import os
import argparse
import sys
import subprocess
import shutil
from typing import Optional

def repair_environment():
    """Install compatible versions of required packages"""
    print("🛠️ Installing required packages...")
    try:
        # Install transformers first to get compatible tokenizers
        subprocess.check_call([
            sys.executable, "-m", "pip", "install", 
            "--upgrade",
            "transformers>=4.40.0,<4.47.0"
        ])
        # Then install the rest
        subprocess.check_call([
            sys.executable, "-m", "pip", "install", 
            "--upgrade",
            "torch>=2.0.0",
            "optimum[onnxruntime]>=1.20.0", 
            "onnx>=1.15.0",
            "peft>=0.10.0",
            "accelerate>=0.20.0"
        ])
        print("✅ Packages installed successfully. Restarting...")
        os.environ["ENV_REPAIRED"] = "1"
        os.execv(sys.executable, [sys.executable] + sys.argv)
    except Exception as e:
        print(f"❌ Failed to install packages: {e}")
        print("\n💡 Try installing manually:")
        print("pip install 'transformers>=4.40.0,<4.47.0' torch 'optimum[onnxruntime]' onnx peft accelerate")
        sys.exit(1)

# Check if --skip-env-check flag is present
skip_env_check = "--skip-env-check" in sys.argv
if skip_env_check:
    sys.argv.remove("--skip-env-check")
    print("⏭️ Skipping environment check as requested")
else:
    if "ENV_REPAIRED" not in os.environ:
        try:
            import torch
            from transformers import AutoModelForCausalLM, AutoTokenizer
            from peft import PeftModel
            from optimum.exporters.onnx import main_export
            print("✅ All required packages are already installed")
        except ImportError as e:
            print(f"⚠️ Missing package: {e}")
            print("Installing required packages...")
            repair_environment()

# Now import everything
import torch
from transformers import AutoModelForCausalLM, AutoTokenizer, AutoConfig
from peft import PeftModel
from optimum.exporters.onnx import OnnxConfig
from optimum.exporters.onnx.model_configs import GemmaOnnxConfig
from optimum.utils import DEFAULT_DUMMY_SHAPES
import onnx

# Custom ONNX configuration for Gemma3/FunctionGemma
class Gemma3OnnxConfig(GemmaOnnxConfig):
    """Custom ONNX configuration for Gemma3 models (including FunctionGemma)"""
    
    DEFAULT_ONNX_OPSET = 14
    NORMALIZED_CONFIG_CLASS = "gemma3"
    
    def __init__(
        self,
        config,
        task: str = "causal-lm",
        int_dtype: str = "int64",
        float_dtype: str = "fp32",
        use_past: bool = False,
        use_past_in_inputs: bool = False,
        preprocessors = None,
        legacy: bool = False,
    ):
        super().__init__(
            config,
            task=task,
            int_dtype=int_dtype,
            float_dtype=float_dtype,
            use_past=use_past,
            use_past_in_inputs=use_past_in_inputs,
            preprocessors=preprocessors,
            legacy=legacy,
        )

def convert_to_onnx_manual(model_path: str, output_path: str, use_past: bool = False):
    """Manually convert model to ONNX using legacy torch.onnx.export"""
    print(f"🚀 Converting model from {model_path} to ONNX format...")
    print(f"⚠️ Using legacy ONNX export method for gemma3_text architecture")
    
    try:
        # Load model and tokenizer
        print("📥 Loading model...")
        config = AutoConfig.from_pretrained(model_path, trust_remote_code=True)
        model = AutoModelForCausalLM.from_pretrained(
            model_path,
            torch_dtype=torch.float32,
            device_map="cpu",
            trust_remote_code=True,
            low_cpu_mem_usage=True
        )
        model.eval()
        
        tokenizer = AutoTokenizer.from_pretrained(model_path, trust_remote_code=True)
        
        # Create dummy inputs
        print("🔧 Creating dummy inputs...")
        batch_size = 1
        seq_length = 32  # Use shorter sequence for faster export
        
        dummy_input_ids = torch.randint(0, config.vocab_size, (batch_size, seq_length), dtype=torch.long)
        dummy_attention_mask = torch.ones((batch_size, seq_length), dtype=torch.long)
        
        # Create output directory
        os.makedirs(output_path, exist_ok=True)
        
        # Export to ONNX using legacy exporter
        output_file = os.path.join(output_path, "model.onnx")
        print(f"📤 Exporting to ONNX (this may take several minutes)...")
        print(f"💡 Using legacy exporter to avoid torch.export issues...")
        
        # Disable problematic attention implementations for export
        # Force eager attention (no SDPA which causes tracing issues)
        print("💡 Configuring model for ONNX export compatibility...")
        if hasattr(model.config, '_attn_implementation'):
            original_attn = model.config._attn_implementation
            model.config._attn_implementation = 'eager'
        
        # Wrap model to return only logits and handle attention mask properly
        class ModelWrapper(torch.nn.Module):
            def __init__(self, model):
                super().__init__()
                self.model = model
            
            def forward(self, input_ids, attention_mask):
                # Use simple attention mask without complex masking
                outputs = self.model(
                    input_ids=input_ids, 
                    attention_mask=attention_mask,
                    use_cache=False  # Disable KV cache for simpler export
                )
                return outputs.logits
        
        wrapped_model = ModelWrapper(model)
        wrapped_model.eval()
        
        with torch.no_grad():
            # Use legacy exporter explicitly
            torch.onnx.export(
                wrapped_model,
                (dummy_input_ids, dummy_attention_mask),
                output_file,
                export_params=True,
                input_names=['input_ids', 'attention_mask'],
                output_names=['logits'],
                dynamic_axes={
                    'input_ids': {0: 'batch_size', 1: 'sequence_length'},
                    'attention_mask': {0: 'batch_size', 1: 'sequence_length'},
                    'logits': {0: 'batch_size', 1: 'sequence_length'}
                },
                opset_version=17,  # Use newer opset
                do_constant_folding=True,
                verbose=False,
                # Use legacy exporter
                dynamo=False
            )
        
        # Save tokenizer and config
        print("💾 Saving tokenizer and config...")
        tokenizer.save_pretrained(output_path)
        config.save_pretrained(output_path)
        
        # Verify the ONNX model
        print("✅ Verifying ONNX model...")
        onnx_model = onnx.load(output_file)
        onnx.checker.check_model(onnx_model)
        
        print(f"✅ Conversion successful! Output at: {output_path}")
        print(f"📄 ONNX model file: {output_file}")
        
        # Clean up
        del model
        import gc
        gc.collect()
        
        return True
        
    except Exception as e:
        print(f"❌ Error during conversion: {str(e)}")
        import traceback
        traceback.print_exc()
        return False

def convert_with_optimum(model_path: str, output_path: str):
    """Try to convert using Optimum with custom config"""
    print(f"🚀 Attempting conversion with Optimum + custom config...")
    
    try:
        from optimum.exporters.onnx import export
        
        # Load config to create custom ONNX config
        config = AutoConfig.from_pretrained(model_path, trust_remote_code=True)
        
        # Create custom ONNX config
        onnx_config = Gemma3OnnxConfig(
            config=config,
            task="text-generation-with-past",
            use_past=True
        )
        
        # Load model
        model = AutoModelForCausalLM.from_pretrained(
            model_path,
            torch_dtype=torch.float32,
            device_map="cpu",
            trust_remote_code=True,
            low_cpu_mem_usage=True
        )
        model.eval()
        
        # Export
        export(
            model=model,
            config=onnx_config,
            output=output_path,
        )
        
        print(f"✅ Conversion successful! Output at: {output_path}")
        return True
        
    except Exception as e:
        print(f"⚠️ Optimum conversion failed: {str(e)}")
        return False

def main():
    parser = argparse.ArgumentParser(
        description="Convert fine-tuned FunctionGemma to ONNX for WebGPU (Local CPU)",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  # Convert a LoRA adapter model
  python script.py --model_id "google/gemma-2b" --finetuned_model_path "./my_lora"
  
  # Convert and skip environment check
  python script.py --model_id "google/gemma-2b" --finetuned_model_path "./my_lora" --skip-env-check
  
  # Use manual conversion method
  python script.py --model_id "google/gemma-2b" --finetuned_model_path "./my_model" --force-manual
        """
    )
    parser.add_argument("--model_id", type=str, required=True, 
                        help="Base model ID (HuggingFace model name)")
    parser.add_argument("--finetuned_model_path", type=str, required=True,
                        help="Local path to fine-tuned model directory")
    parser.add_argument("--output_dir", type=str, default="./onnx_output",
                        help="Output directory for ONNX model (default: ./onnx_output)")
    parser.add_argument("--skip-env-check", action="store_true",
                        help="Skip automatic environment check and package installation")
    parser.add_argument("--force-manual", action="store_true",
                        help="Force manual ONNX conversion (bypass Optimum)")
    
    args = parser.parse_args()
    
    # Validate that the model path exists
    if not os.path.exists(args.finetuned_model_path):
        raise ValueError(f"❌ Model path does not exist: {args.finetuned_model_path}")
    
    print(f"\n{'='*60}")
    print(f"🤖 Model Conversion Configuration")
    print(f"{'='*60}")
    print(f"Base Model ID: {args.model_id}")
    print(f"Fine-tuned Model: {args.finetuned_model_path}")
    print(f"Output Directory: {args.output_dir}")
    print(f"Device: CPU")
    print(f"Conversion Method: {'Manual' if args.force_manual else 'Auto (Optimum + Manual fallback)'}")
    print(f"{'='*60}\n")
    
    model_to_export_path = args.finetuned_model_path
    temp_merged_dir = "./temp_merged_model"
    
    is_lora = os.path.exists(os.path.join(args.finetuned_model_path, "adapter_config.json"))
    
    if is_lora:
        print("ℹ️ LoRA adapter detected. Merging with base model...")
        
        print("📥 Loading tokenizer...")
        tokenizer = AutoTokenizer.from_pretrained(args.model_id, trust_remote_code=True)
        
        print("📥 Loading base model on CPU (this may take a while)...")
        base_model = AutoModelForCausalLM.from_pretrained(
            args.model_id,
            torch_dtype=torch.float32,
            device_map="cpu",
            trust_remote_code=True,
            low_cpu_mem_usage=True
        )
        
        print("📥 Loading LoRA adapter...")
        model = PeftModel.from_pretrained(base_model, args.finetuned_model_path)
        
        print("🔄 Merging LoRA adapter with base model...")
        merged_model = model.merge_and_unload()
        
        print(f"💾 Saving merged model to {temp_merged_dir}...")
        os.makedirs(temp_merged_dir, exist_ok=True)
        merged_model.save_pretrained(temp_merged_dir, safe_serialization=True)
        tokenizer.save_pretrained(temp_merged_dir)
        model_to_export_path = temp_merged_dir
        
        # Clean up memory
        print("🧹 Cleaning up memory...")
        del base_model
        del model
        del merged_model
        import gc
        gc.collect()
    else:
        print("ℹ️ Full model detected (not a LoRA adapter).")
        # Ensure tokenizer files exist
        if not os.path.exists(os.path.join(args.finetuned_model_path, "tokenizer_config.json")):
            print("⚠️ Tokenizer not found in model directory. Downloading from base model...")
            tokenizer = AutoTokenizer.from_pretrained(args.model_id, trust_remote_code=True)
            tokenizer.save_pretrained(args.finetuned_model_path)
        else:
            print("✅ Tokenizer found in model directory.")

    # Create output directory
    os.makedirs(args.output_dir, exist_ok=True)
    
    # Convert to ONNX
    print(f"\n{'='*60}")
    print("🔧 Starting ONNX conversion")
    print("⏱️ This may take several minutes on CPU...")
    print(f"{'='*60}\n")
    
    success = False
    
    # Try conversion methods
    if not args.force_manual:
        print("📍 Trying Optimum with custom configuration first...")
        success = convert_with_optimum(model_to_export_path, args.output_dir)
    
    if not success:
        print("\n📍 Using manual PyTorch ONNX export...")
        success = convert_to_onnx_manual(model_to_export_path, args.output_dir)
    
    # Clean up temporary merged directory
    if os.path.exists(temp_merged_dir) and temp_merged_dir != args.finetuned_model_path:
        print(f"\n🧹 Cleaning up temporary directory: {temp_merged_dir}")
        shutil.rmtree(temp_merged_dir)
    
    if success:
        print(f"\n{'='*60}")
        print("🎉 Conversion completed successfully!")
        print(f"{'='*60}")
        print(f"📂 ONNX model location: {os.path.abspath(args.output_dir)}")
        print(f"📄 Main model file: {os.path.join(args.output_dir, 'model.onnx')}")
        print(f"{'='*60}\n")
    else:
        print(f"\n{'='*60}")
        print("❌ Conversion failed!")
        print(f"{'='*60}\n")
        sys.exit(1)

if __name__ == "__main__":
    main()