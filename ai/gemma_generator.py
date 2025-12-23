import json
import os
from transformers import AutoTokenizer
from training.tools_config import SYSTEM_INSTRUCTION

def generate_training_data(data_configs):
    """
    Prepares the training data in a structured message format.
    The actual formatting with tools will be handled during training
    to ensure consistency with inference.
    """
    processed_data = []
    
    # Base system instruction from shared config
    system_instruction = SYSTEM_INSTRUCTION

    for config in data_configs:
        assistant_template = config.get("assistant")
        queries = config.get("queries", [])
        
        for query_info in queries:
            user_query = query_info.get("user")
            variables = query_info.get("variables", [])
            
            # Replace [VARIABLE_N] placeholders in assistant template
            assistant_content = assistant_template
            for i, var_value in enumerate(variables):
                placeholder = f"[VARIABLE_{i+1}]"
                # Handle null/empty variables
                val = str(var_value) if var_value is not None else "null"
                assistant_content = assistant_content.replace(placeholder, val)
            
            # Reduce double braces to single braces for the final call
            assistant_content = assistant_content.replace("{{", "{").replace("}}", "}")

            # Create message structure
            # Note: We keep the content "clean" (no manual tags) 
            # because the trainer will apply the tools during training.
            example = {
                "messages": [
                    {
                        "role": "developer",
                        "content": system_instruction
                    },
                    {
                        "role": "user",
                        "content": user_query
                    },
                    {
                        "role": "assistant",
                        "content": assistant_content
                    }
                ]
            }
            processed_data.append(example)
            
    return processed_data

def save_jsonl(data, filename):
    with open(filename, 'w', encoding='utf-8') as f:
        for entry in data:
            f.write(json.dumps(entry, ensure_ascii=False) + '\n')

if __name__ == "__main__":
    from gemma_training_data import training_data
    
    print("Generating training examples...")
    processed_examples = generate_training_data(training_data)
    save_jsonl(processed_examples, "gemma_training_data.jsonl")
    print(f"Generated {len(processed_examples)} examples in gemma_training_data.jsonl")
