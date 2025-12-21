import json

def generate_training_data(data_configs):
    """
    Generates training data for functionGemma from a list of configurations.
    
    Each config should have:
    - developer: The system instruction with function declaration.
    - assistant: A template for the assistant response with [VARIABLE_N] placeholders.
    - queries: A list of dicts with 'user' (query) and 'variables' (list of values).
    """
    training_data = []
    
    for config in data_configs:
        developer_content = config.get("developer")
        assistant_template = config.get("assistant")
        queries = config.get("queries", [])
        
        for query_info in queries:
            user_query = query_info.get("user")
            variables = query_info.get("variables", [])
            
            # Replace [VARIABLE_N] placeholders in assistant template
            # [VARIABLE_1] -> variables[0], [VARIABLE_2] -> variables[1], etc.
            assistant_content = assistant_template
            for i, var_value in enumerate(variables):
                placeholder = f"[VARIABLE_{i+1}]"
                assistant_content = assistant_content.replace(placeholder, str(var_value))
            
            # Ensure double braces in template (if any) are reduced to single braces
            # if the user provided something like {{producto: ... }}
            # but wait, the user's prompt has {{ which might be literal or for formatting.
            # In the prompt: assistant: "<start_function_call>call:get_almacen_stock{{producto:<escape>[VARIABLE_1]<escape>}}<end_function_call>"
            # If we want the output to be: <start_function_call>call:get_almacen_stock{producto:<escape>value<escape>}<end_function_call>
            # we should replace {{ with { and }} with }.
            assistant_content = assistant_content.replace("{{", "{").replace("}}", "}")

            example = {
                "messages": [
                    {
                        "role": "developer",
                        "content": developer_content
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
            training_data.append(example)
            
    return training_data

def save_jsonl(data, filename):
    with open(filename, 'w', encoding='utf-8') as f:
        for entry in data:
            f.write(json.dumps(entry, ensure_ascii=False) + '\n')

if __name__ == "__main__":
    from gemma_training_data import training_data
    
    training_data = generate_training_data(training_data)
    save_jsonl(training_data, "gemma_training_data.jsonl")
    print(f"Generated {len(training_data)} examples in gemma_training_data.jsonl")

