import os
import argparse
import subprocess
import sys
from transformers import AutoTokenizer
from training.tools_config import ALL_TOOLS, SYSTEM_INSTRUCTION

# --- llama-cpp-python installation note ---
# To use Vulkan with llama-cpp-python, you must install it with the Vulkan flag:
# CMAKE_ARGS="-DGGML_VULKAN=1" pip install llama-cpp-python --upgrade --force-reinstall --no-cache-dir
# ------------------------------------------

try:
    from llama_cpp import Llama
except ImportError:
    print("llama-cpp-python not found. Installing...")
    # Attempting to install with Vulkan support
    env = os.environ.copy()
    env["CMAKE_ARGS"] = "-DGGML_VULKAN=1"
    subprocess.check_call([sys.executable, "-m", "pip", "install", "llama-cpp-python"], env=env)
    from llama_cpp import Llama

def run_inference(user_query, gguf_path=None):
    if gguf_path is None:
        gguf_path = "/home/ivanjoz/projects/genix/ai/models/functiongemma_270m_finetuned.gguf"
    
    # Path to tokenizer (we use this to apply the chat template correctly)
    tokenizer_path = "/home/ivanjoz/projects/genix/ai/models/functiongemma_270m_finetuned"
    
    if not os.path.exists(gguf_path):
        print(f"ERROR: GGUF model not found at {gguf_path}")
        print("Please run models/convert_to_gguf.py first.")
        sys.exit(1)

    print(f"Loading GGUF model from {gguf_path}...")
    print("Using VULKAN acceleration (n_gpu_layers=-1)")
    
    # 1. Initialize Llama with Vulkan
    # n_gpu_layers=-1 moves all layers to Vulkan GPU
    llm = Llama(
        model_path=gguf_path,
        n_gpu_layers=-1, 
        n_ctx=2048,
        verbose=False
    )

    # 2. Prepare prompt with Chat Template (using Transformers tokenizer for consistency)
    print(f"Loading tokenizer from {tokenizer_path} for prompt formatting...")
    tokenizer = AutoTokenizer.from_pretrained(tokenizer_path, trust_remote_code=True)
    
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

    print("\n--- Generating Response (llama.cpp + Vulkan) ---")
    print(f"Query: {user_query}")

    # 3. Generate
    output = llm(
        prompt,
        max_tokens=128,
        stop=["<end_of_turn>", "<eos>"],
        echo=False,
        temperature=0.0 # Deterministic for function calling
    )

    response = output["choices"][0]["text"].strip()
    print(f"Generated Response: {response}")

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Run inference using llama.cpp with Vulkan acceleration")
    parser.add_argument(
        "-m", "--message", 
        type=str, 
        required=True, 
        help="The user question or message to the model"
    )
    parser.add_argument(
        "--model", 
        type=str, 
        default="/home/ivanjoz/projects/genix/ai/models/functiongemma_270m_finetuned.gguf",
        help="Path to the GGUF model file"
    )
    
    args = parser.parse_args()
    
    run_inference(args.message, gguf_path=args.model)
