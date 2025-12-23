import os
import torch
import argparse
from transformers import AutoTokenizer, AutoModelForCausalLM
from peft import PeftModel
from training.tools_config import ALL_TOOLS, SYSTEM_INSTRUCTION

def run_inference(user_query, no_adapter=False, use_full=False):
    # Paths to the model and adapter
    base_model_path = "/home/ivanjoz/projects/genix/ai/models/functiongemma-270m-it"
    adapter_path = "/home/ivanjoz/projects/genix/ai/models/functiongemma_270m_lora"
    full_model_path = "/home/ivanjoz/projects/genix/ai/models/functiongemma_270m_finetuned"

    # 1. Force Vulkan Device
    if not torch.is_vulkan_available():
        print("ERROR: Native Vulkan support not found in PyTorch.")
        print("Make sure you have a Vulkan-enabled PyTorch build and drivers installed.")
        # We don't exit here to let model.to("vulkan") throw a clearer error if it fails
    
    device = torch.device("vulkan")
    print(f"FORCING DEVICE: {device}")

    # 2. Determine load path
    if use_full:
        load_path = full_model_path
        print(f"Mode: FULL FINETUNED MODEL")
    elif no_adapter:
        load_path = base_model_path
        print(f"Mode: BASE MODEL (No LoRA)")
    else:
        load_path = adapter_path
        print(f"Mode: LoRA ADAPTER")

    # 3. Load Tokenizer
    tokenizer_path = full_model_path if use_full else (adapter_path if not no_adapter else base_model_path)
    print(f"Loading tokenizer from {tokenizer_path}...")
    tokenizer = AutoTokenizer.from_pretrained(tokenizer_path, trust_remote_code=True)

    # 4. Load Model
    print("Loading model in PyTorch...")
    if use_full:
        model = AutoModelForCausalLM.from_pretrained(
            full_model_path,
            trust_remote_code=True,
            torch_dtype=torch.float32 # Vulkan support for float16 varies
        )
    else:
        model = AutoModelForCausalLM.from_pretrained(
            base_model_path,
            trust_remote_code=True,
            torch_dtype=torch.float32
        )
        if not no_adapter:
            print(f"Loading LoRA adapter from {adapter_path}...")
            model = PeftModel.from_pretrained(model, adapter_path)

    # 5. Move to Vulkan
    print(f"Moving model to {device}...")
    try:
        model.to(device)
    except Exception as e:
        print(f"FATAL: Could not move model to Vulkan: {e}")
        exit(1)

    # 6. Prepare prompt with Chat Template
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

    prompt = tokenizer.apply_chat_template(
        messages,
        tools=ALL_TOOLS,
        add_generation_prompt=True,
        tokenize=False
    )

    # 7. Tokenize and Move Inputs to Vulkan
    inputs = tokenizer(prompt, return_tensors="pt").to(device)

    print("\n--- Generating Response ---")
    print(f"Query: {user_query}")

    # 8. Generate
    try:
        outputs = model.generate(
            **inputs,
            max_new_tokens=128,
            do_sample=False
        )
    except Exception as e:
        print(f"\nFATAL GENERATION ERROR: {e}")
        raise e

    # 9. Decode only the new tokens
    generated_ids = outputs[0][len(inputs.input_ids[0]):]
    response = tokenizer.decode(generated_ids, skip_special_tokens=True)

    print(f"Generated Response: {response}")

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Run inference with LoRA or Full Finetuned model strictly on Vulkan")
    parser.add_argument(
        "-m", "--message", 
        type=str, 
        required=True, 
        help="The user question or message to the model"
    )
    parser.add_argument(
        "--base", 
        action="store_true", 
        help="Use bare base model (No LoRA adapter) for comparison"
    )
    parser.add_argument(
        "--full", 
        action="store_true", 
        help="Use the FULL fine-tuned model"
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
            
    run_inference(args.message, no_adapter=args.base, use_full=args.full)
