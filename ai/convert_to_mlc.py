import os
import json
import subprocess
import sys

# --- CONFIGURATION ---
MODEL_PATH = "./models/functiongemma_270m_finetuned"  
OUTPUT_PATH = "./models/functiongemma_270m_finetuned/mlc"
QUANTIZATION = "q0f16" # Good for WebGPU
CONV_TEMPLATE = "gemma3_instruction"

def run_command(command):
    print(f"🚀 Running: {' '.join(command)}")
    try:
        subprocess.run(command, check=True)
    except subprocess.CalledProcessError as e:
        print(f"❌ Error running command: {e}")
        sys.exit(1)

def main():
    # 0. Check if mlc_llm is installed
    try:
        import mlc_llm
        print("✅ MLC LLM found.")
    except ImportError:
        print("❌ MLC LLM not found. Please run:")
        print("pip install --pre -U -f https://mlc.ai/wheels mlc-ai-nightly-cpu mlc-llm-nightly-cpu")
        sys.exit(1)

    # 1. Create Output Directory
    if not os.path.exists(OUTPUT_PATH):
        os.makedirs(OUTPUT_PATH)

    # 2. Convert Model Weights
    # This creates the sharded binary files
    run_command([
        sys.executable, "-m", "mlc_llm", "convert_weight",
        MODEL_PATH,
        "--quantization", QUANTIZATION,
        "-o", OUTPUT_PATH
    ])

    # 3. Generate Base Configuration
    run_command([
        sys.executable, "-m", "mlc_llm", "gen_config",
        MODEL_PATH,
        "--quantization", QUANTIZATION,
        "--conv-template", CONV_TEMPLATE,
        "-o", OUTPUT_PATH
    ])

    # 4. Inject Special FunctionGemma Configuration
    config_file = os.path.join(OUTPUT_PATH, "mlc-chat-config.json")
    if os.path.exists(config_file):
        with open(config_file, "r") as f:
            config = json.load(f)

        # Apply the Developer/Tool-use template for FunctionGemma
        config["conv_template"] = {
          "name": "gemma3_instruction",
          "system_template": "<start_of_turn>developer\n{system_message}",
          "system_message": "",
          "system_prefix_token_ids": [
            2
          ],
          "add_role_after_system_message": True,
          "roles": {
            "user": "<start_of_turn>user",
            "assistant": "<start_of_turn>model"
          },
          "role_templates": {
            "user": "{user_message}",
            "assistant": "{assistant_message}",
            "tool": "{tool_message}"
          },
          "messages": [],
          "seps": [
            "<end_of_turn>\n"
          ],
          "role_content_sep": "\n",
          "role_empty_sep": "\n",
          "stop_str": [
            "<end_of_turn>"
          ],
          "stop_token_ids": [
            1,
            106
          ],
          "function_string": "",
          "use_function_calling": False
        }
        
        # FunctionGemma 270M optimization
        config["context_window_size"] = 8192
        config["temperature"] = 1.0
        config["eos_token_id"] = [1, 50, 106]
        config["pad_token_id"] = 0

        with open(config_file, "w") as f:
            json.dump(config, f, indent=2)
        print(f"🛠️ Config patched for FunctionGemma.")

    print(f"\n✨ Done! Converted model is at: {OUTPUT_PATH}")

if __name__ == "__main__":
    main()