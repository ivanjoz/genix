import json
import torch
import os
import warnings
from transformers import AutoTokenizer, logging
from optimum.intel import OVModelForCausalLM
from training.tools_config import ALL_TOOLS, SYSTEM_INSTRUCTION

# Suppress warnings for cleaner output
warnings.filterwarnings("ignore")
logging.set_verbosity_error()

def validate():
    # Paths (using absolute paths as preferred)
    full_model_path = "/home/ivanjoz/projects/genix/ai/models/functiongemma_270m_finetuned"
    training_data_path = "/home/ivanjoz/projects/genix/ai/gemma_training_data.jsonl"
    # Match the path used in inference.py for optimized GPU model
    ov_model_dir = "/home/ivanjoz/projects/genix/ai/models/ov_functiongemma_270m_gpu_full"

    if not os.path.exists(full_model_path):
        print(f"Error: Full model path not found: {full_model_path}")
        return

    print(f"Loading tokenizer from {full_model_path}...")
    tokenizer = AutoTokenizer.from_pretrained(full_model_path, trust_remote_code=True)
    
    device = "GPU"
    print(f"Loading OpenVINO model on {device}...")
    
    try:
        if os.path.exists(ov_model_dir):
            print(f"Loading pre-optimized model from {ov_model_dir}...")
            model = OVModelForCausalLM.from_pretrained(ov_model_dir, device=device)
        else:
            print(f"Exporting model to OpenVINO (GPU)...")
            model = OVModelForCausalLM.from_pretrained(
                full_model_path,
                export=True,
                device=device,
                trust_remote_code=True
            )
            # Optionally save it for next time
            # model.save_pretrained(ov_model_dir)
        
        print("SUCCESS: Model is running on accelerated GPU")
    except Exception as e:
        print(f"Error loading OpenVINO GPU: {e}. Falling back to CPU...")
        device = "CPU"
        model = OVModelForCausalLM.from_pretrained(
            full_model_path,
            export=True,
            device=device,
            trust_remote_code=True
        )

    print(f"Loading training data from {training_data_path}...")
    if not os.path.exists(training_data_path):
        print(f"Error: Training data not found: {training_data_path}")
        return
        
    examples = []
    with open(training_data_path, "r", encoding="utf-8") as f:
        for line in f:
            examples.append(json.loads(line))

    print(f"Validating {len(examples)} examples...\n")
    
    correct_count = 0
    
    for i, example in enumerate(examples):
        # Extract the user query and expected assistant response
        messages = example["messages"]
        user_query = ""
        expected_response = ""
        
        for msg in messages:
            if msg["role"] == "user":
                user_query = msg["content"]
            elif msg["role"] == "assistant":
                expected_response = msg["content"]

        # Prepare the inference prompt
        inference_messages = [
            {"role": "developer", "content": SYSTEM_INSTRUCTION},
            {"role": "user", "content": user_query}
        ]

        prompt = tokenizer.apply_chat_template(
            inference_messages,
            tools=ALL_TOOLS,
            add_generation_prompt=True,
            tokenize=False
        )

        inputs = tokenizer(prompt, return_tensors="pt")
        
        # Generate response
        try:
            outputs = model.generate(
                **inputs,
                max_new_tokens=128,
                do_sample=False
            )
            
            # Decode only the new tokens
            generated_ids = outputs[0][len(inputs.input_ids[0]):]
            generated_response = tokenizer.decode(generated_ids, skip_special_tokens=True).strip()
            
            # Verification logic
            is_correct = generated_response == expected_response.strip()
            
            # Fallback for special tokens if needed
            if not is_correct:
                raw_generated = tokenizer.decode(generated_ids, skip_special_tokens=False).strip()
                if "<eos>" in raw_generated:
                    raw_generated = raw_generated.split("<eos>")[0].strip()
                if raw_generated == expected_response.strip():
                    generated_response = raw_generated
                    is_correct = True

            if is_correct:
                correct_count += 1
            
            print(f"[{i+1}/{len(examples)}] {'✓' if is_correct else '✗'} Query: {user_query}", flush=True)
            print(f"   Generated: {generated_response}", flush=True)
            if not is_correct:
                print(f"   Expected:  {expected_response}", flush=True)
            print("-" * 20, flush=True)
            
        except Exception as e:
            print(f"[{i+1}/{len(examples)}] ERROR generating for: {user_query}")
            print(f"   Error: {e}")
            print("-" * 20)

    print("\n" + "="*30)
    print("VALIDATION SUMMARY")
    print("="*30)
    print(f"Total Examples: {len(examples)}")
    print(f"Correct:        {correct_count}")
    print(f"Incorrect:      {len(examples) - correct_count}")
    print(f"Accuracy:       {correct_count / len(examples) * 100:.2f}%")
    print("="*30)


if __name__ == "__main__":
    validate()