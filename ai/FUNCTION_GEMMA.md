# FUNCTION_GEMMA.md

## 1. Overview
**FunctionGemma** is a lightweight, open-source specialized version of the **Gemma 3 270M** model from Google. While most LLMs are general-purpose, FunctionGemma is a "narrow specialist" designed specifically to act as a bridge between natural language and executable code (APIs). 

Its primary purpose is to run on-device (Web, Mobile, Edge) and convert a user's intent into a structured **JSON-like function call** using a specific set of proprietary control tokens. It is built using the same research and technology behind Gemini models.

### 1.1 Technical Specifications
* **Architecture:** Based on Gemma 3 270M.
* **Context Window:** 32,768 (32K) tokens for both input and output.
* **Parameter Count:** ~270 Million (optimized for edge devices).
* **Training Data:** 6 Trillion tokens, including public tool definitions and tool-use interactions.
* **Knowledge Cutoff:** August 2024.
* **Intended Use:** Fine-tuning for specific function-calling tasks and agentic workflows. It is **not** intended for direct dialogue.

## 2. Conversation Architecture
FunctionGemma follows a strict turn-based system. Each turn is wrapped in `<start_of_turn>` and `<end_of_turn>` tags.

| Role | Responsibility |
| :--- | :--- |
| **developer** | Defines the "System Instruction" and the list of available functions (tools). |
| **user** | The natural language query from the human. |
| **model** | The structured function call (the "thought" of the model). |



## 3. The Control Tokens
Unlike standard models that use Markdown code blocks, FunctionGemma uses these specialized tokens to prevent "hallucinations" and simplify parsing:

* `<start_function_declaration>` / `<end_function_declaration>`: Wraps the API definition.
* `<start_function_call>` / `<end_function_call>`: Wraps the model's output.
* `<escape>`: Wraps every single **string value** inside the JSON to protect special characters.
* `call:`: A prefix that must appear immediately after the start call token.

## 4. Training Data Structure (JSONL)
To train the model, we use a JSONL format where each object is a `messages` array. **Note the specific string required in the developer role to "activate" the model.**

### Example: Spanish Weather Function
```json
{
  "messages": [
    {
      "role": "developer",
      "content": "You are a model that can do function calling with the following functions<start_function_declaration>declaration:get_weather{description:<escape>Obtener el clima de una ciudad<escape>,parameters:{properties:{city:{type:<escape>STRING<escape>}},required:[<escape>city<escape>]}}<end_function_declaration>"
    },
    {
      "role": "user",
      "content": "¿Qué tiempo hace en Madrid?"
    },
    {
      "role": "assistant",
      "content": "<start_function_call>call:get_weather{city:<escape>Madrid<escape>}<end_function_call>"
    }
  ]
}

## 5. Instructions for Data Generation
When generating synthetic data for FunctionGemma, the following logic must be strictly applied:

1.  **Escape Strings:** Every value that is a string MUST be wrapped in the `<escape>` token. 
    * *Correct:* `city:<escape>Madrid<escape>`
    * *Incorrect:* `city:"Madrid"`
2.  **No Extra Text:** The assistant's response in training data should ONLY contain the function call tokens. Do not include conversational filler like "Claro, aquí tienes" or "Entendido".
3.  **Schema Formatting:** The function declaration inside the developer turn must use the `declaration:name{...}` format.
4.  **Language Robustness:** For Spanish instructions, generate variations including:
    * **Direct:** "Clima en Barcelona"
    * **Polite:** "Por favor, dime el tiempo de Valencia"
    * **Colloquial:** "¿Cómo va el día en Sevilla?"
    * **Typo-prone:** "Tiempo en Madrir" (helps the model learn to handle messy input).
    * **Implicit:** "¿Necesito llevar paraguas en Bogotá?" (the model must infer this means `get_weather`).
5.  **Output Format:** Each example must be a single line (JSONL) to be compatible with training scripts.

## 6. Python Implementation (Hugging Face)
To use FunctionGemma in a Python environment, you must use the `transformers` library with the specific `AutoProcessor` that handles the custom chat templates and control tokens.

### Installation
```bash
pip install torch transformers
```

### Basic Usage
```python
from transformers import AutoProcessor, AutoModelForCausalLM

# Load model and processor
model_id = "google/functiongemma-270m-it"
processor = AutoProcessor.from_pretrained(model_id)
model = AutoModelForCausalLM.from_pretrained(model_id, device_map="auto", torch_dtype="auto")

# Define the function schema (standard JSON Schema)
weather_tool = {
    "type": "function",
    "function": {
        "name": "get_current_temperature",
        "description": "Gets the current temperature for a given location.",
        "parameters": {
            "type": "object",
            "properties": {
                "location": {"type": "string", "description": "The city name, e.g. San Francisco"}
            },
            "required": ["location"]
        }
    }
}

# Prepare the prompt
messages = [
    {"role": "developer", "content": "You are a model that can do function calling with the following functions"},
    {"role": "user", "content": "What's the temperature in London?"}
]

# Apply chat template with tools
inputs = processor.apply_chat_template(
    messages, 
    tools=[weather_tool], 
    add_generation_prompt=True, 
    return_dict=True, 
    return_tensors="pt"
).to(model.device)

# Generate
out = model.generate(**inputs, max_new_tokens=128)
output = processor.decode(out[0][len(inputs["input_ids"][0]):], skip_special_tokens=True)

print(output)
# Output: <start_function_call>call:get_current_temperature{location:<escape>London<escape>}<end_function_call>
```

## 7. Performance & On-Device Deployment
FunctionGemma is optimized for high-speed inference on consumer hardware.

| Metric | Performance (Samsung S25 Ultra) |
| :--- | :--- |
| **Decode Speed** | ~125 tokens per second |
| **Prefill Speed** | ~1700 tokens per second |
| **Model Size** | ~288 MB (int8 quantized) |
| **Peak Memory** | ~550 MB |

### Why use FunctionGemma?
1. **Privacy:** Processes data entirely on-device without cloud connectivity.
2. **Latency:** Extremely low "Time-to-first-token" (approx. 0.3s on mobile).
3. **Efficiency:** Handles complex agentic tasks with a fraction of the parameters of larger models.