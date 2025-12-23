import os
import torch
import argparse
from transformers import AutoTokenizer, AutoModelForCausalLM
from optimum.intel import OVModelForCausalLM
from peft import PeftModel

# --- Installation & Setup (Fedora/Linux with Intel GPU) ---
# 1. System-level setup (Intel Compute Drivers):
#    Fedora requires the Intel compute runtime and Level Zero drivers for AI acceleration.
#    Run: sudo dnf install intel-compute-runtime intel-level-zero oneapi-level-zero
#
# 2. Permissions setup:
#    To allow your user to access the GPU without sudo, add yourself to the render/video groups.
#    Run: sudo usermod -aG video,render $USER
#    CRITICAL: Log out and back in, OR run 'newgrp render' in your current terminal.
#
# 3. Virtual Environment setup:
#    python3.12 -m venv venv (Note: Python 3.12 is recommended for best driver compatibility)
#    source venv/bin/activate
#
# 4. Install Intel-optimized libraries:
#    Run: pip install --upgrade pip
#    Run: pip install "optimum[openvino]" transformers peft accelerate
#
# WHY OPENVINO?
# This script uses OpenVINO instead of native PyTorch (XPU/Vulkan) because:
# - STABILITY: The experimental PyTorch XPU backend currently has driver bugs on Linux 
#   (e.g., UR error: 45) with new models like Gemma 3.
# - OPTIMIZATION: OpenVINO is Intel's official high-performance engine.
# - COMPATIBILITY: It handles complex features like "Sliding Window Attention" 
#   much more reliably on Iris Xe/Arc hardware.
#
# 5. (Optional) Verify GPU visibility:
#    Run: sudo dnf install igt-gpu-tools
#    Run 'sudo intel_gpu_top' in a separate terminal to watch the GPU activity.
# -----------------------------

def run_inference(user_query, use_cpu=False, use_raw=False, no_adapter=False, use_full=False):
    # Paths to the model and adapter
    base_model_path = "/home/ivanjoz/projects/genix/ai/models/functiongemma-270m-it"
    adapter_path = "/home/ivanjoz/projects/genix/ai/models/functiongemma_270m_lora"
    full_model_path = "/home/ivanjoz/projects/genix/ai/models/functiongemma_270m_finetuned"

    # Determine load path for tokenizer and suffix for OpenVINO
    if use_full:
        load_path = full_model_path
        suffix = "full"
        print(f"Mode: FULL FINETUNED MODEL")
    elif no_adapter:
        load_path = base_model_path
        suffix = "base"
        print(f"Mode: BASE MODEL (No LoRA)")
    else:
        load_path = adapter_path
        suffix = "lora"
        print(f"Mode: LoRA ADAPTER")

    print(f"Loading tokenizer from {load_path}...")
    tokenizer = AutoTokenizer.from_pretrained(load_path, trust_remote_code=True)
    
    device = "CPU" # Default for display
    
    if use_raw:
        print(f"Using RAW PyTorch on CPU (No OpenVINO)... [{suffix.upper()}]")
        # Load model in PyTorch
        if use_full:
            model = AutoModelForCausalLM.from_pretrained(
                full_model_path,
                trust_remote_code=True,
                torch_dtype=torch.float32,
                device_map="cpu"
            )
        else:
            model = AutoModelForCausalLM.from_pretrained(
                base_model_path,
                trust_remote_code=True,
                torch_dtype=torch.float32,
                device_map="cpu"
            )
            if not no_adapter:
                model = PeftModel.from_pretrained(model, adapter_path)
    else:
        # Device selection based on flag
        device = "CPU" if use_cpu else "GPU"
        print(f"Using OpenVINO device: {device} [{suffix.upper()}]")

        # Separate directories for base, full and lora models
        ov_model_dir = f"/home/ivanjoz/projects/genix/ai/models/ov_functiongemma_270m_{device.lower()}_{suffix}"
        
        try:
            if not os.path.exists(ov_model_dir):
                print(f"Exporting model to OpenVINO and saving to {ov_model_dir}...")
                
                if use_full:
                    # Export full model directly
                    model = OVModelForCausalLM.from_pretrained(
                        full_model_path,
                        export=True,
                        device=device,
                        trust_remote_code=True
                    )
                elif no_adapter:
                    # Export base model directly
                    model = OVModelForCausalLM.from_pretrained(
                        base_model_path,
                        export=True,
                        device=device,
                        trust_remote_code=True
                    )
                else:
                    print("(Merging LoRA weights in PyTorch first to ensure they are included)")
                    # Load base model and adapter in PyTorch to merge them
                    base_pytorch = AutoModelForCausalLM.from_pretrained(
                        base_model_path,
                        trust_remote_code=True,
                        torch_dtype=torch.float32
                    )
                    peft_model = PeftModel.from_pretrained(base_pytorch, adapter_path)
                    merged_model = peft_model.merge_and_unload()
                    
                    # Save temporary merged model
                    temp_merged_dir = "/home/ivanjoz/projects/genix/ai/models/temp_merged"
                    merged_model.save_pretrained(temp_merged_dir)
                    tokenizer.save_pretrained(temp_merged_dir)
                    
                    model = OVModelForCausalLM.from_pretrained(
                        temp_merged_dir,
                        export=True,
                        device=device,
                        trust_remote_code=True
                    )
                    
                    import shutil
                    if os.path.exists(temp_merged_dir):
                        shutil.rmtree(temp_merged_dir)
                    del base_pytorch
                    del peft_model
                    del merged_model

                model.save_pretrained(ov_model_dir)
            else:
                print(f"Loading pre-optimized model from {ov_model_dir}...")
                model = OVModelForCausalLM.from_pretrained(ov_model_dir, device=device)
            
            # Verify hardware
            try:
                core = model.request.core
                actual_device = core.get_property(device, "FULL_DEVICE_NAME")
                print(f"SUCCESS: Model is running on: {actual_device}")
            except:
                print(f"SUCCESS: Model is running on accelerated {device}")

        except Exception as e:
            print(f"Error loading OpenVINO: {e}. Falling back to standard PyTorch.")
            if use_full:
                model = AutoModelForCausalLM.from_pretrained(
                    full_model_path,
                    trust_remote_code=True,
                    torch_dtype=torch.float32,
                    device_map="cpu"
                )
            else:
                model = AutoModelForCausalLM.from_pretrained(
                    base_model_path,
                    trust_remote_code=True,
                    torch_dtype=torch.float32,
                    device_map="cpu"
                )
                if not no_adapter:
                    model = PeftModel.from_pretrained(model, adapter_path)
            device = "CPU"

    # Define a sample tool (function) following FunctionGemma format
    from training.tools_config import ALL_TOOLS, SYSTEM_INSTRUCTION
    tools = ALL_TOOLS

    # Prepare the messages for the chat template
    messages = [
        {
            "role": "developer", 
            "content": SYSTEM_INSTRUCTION
        },
        {
            "role": "user", 
            "content": user_query
        }
    ]

    # Apply the chat template
    prompt = tokenizer.apply_chat_template(
        messages,
        tools=tools,
        add_generation_prompt=True,
        tokenize=False
    )

    # Tokenize input
    inputs = tokenizer(prompt, return_tensors="pt")

    print("\n--- Generating Response ---")
    print(f"Query: {user_query}")
    print(f"Device: {device}")

    # Generate
    try:
        outputs = model.generate(
            **inputs,
            max_new_tokens=128,
            do_sample=False
        )
    except Exception as e:
        print(f"\nFATAL GENERATION ERROR: {e}")
        raise e

    # Decode only the new tokens
    generated_ids = outputs[0][len(inputs.input_ids[0]):]
    response = tokenizer.decode(generated_ids, skip_special_tokens=True)

    print(f"Generated Function Call: {response}")

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Run inference with LoRA or Full Finetuned model on Intel GPU/CPU using OpenVINO")
    parser.add_argument(
        "-m", "--message", 
        type=str, 
        required=True, 
        help="The user question or message to the model"
    )
    parser.add_argument(
        "--cpu", 
        action="store_true", 
        help="Use CPU only with OpenVINO"
    )
    parser.add_argument(
        "--raw", 
        action="store_true", 
        help="Use raw PyTorch on CPU (No OpenVINO, best for debugging)"
    )
    parser.add_argument(
        "--base", 
        action="store_true", 
        help="Use bare base model (No LoRA adapter) for comparison"
    )
    parser.add_argument(
        "--full", 
        action="store_true", 
        help="Use the FULL fine-tuned model (from models/functiongemma_270m_finetuned)"
    )
    args = parser.parse_args()
    
    # Path validation
    base_model_path = "/home/ivanjoz/projects/genix/ai/models/functiongemma-270m-it"
    adapter_path = "/home/ivanjoz/projects/genix/ai/models/functiongemma_270m_lora"
    full_model_path = "/home/ivanjoz/projects/genix/ai/models/functiongemma_270m_finetuned"

    if args.full:
        if not os.path.exists(full_model_path):
            print(f"Error: Full finetuned model directory not found at {full_model_path}")
            exit(1)
    elif args.base:
        if not os.path.exists(base_model_path):
            print(f"Error: Base model directory not found at {base_model_path}")
            exit(1)
    else:
        if not os.path.exists(adapter_path):
            print(f"Error: LoRA adapter directory not found at {adapter_path}")
            exit(1)
            
    run_inference(args.message, use_cpu=args.cpu, use_raw=args.raw, no_adapter=args.base, use_full=args.full)
