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
* `declaration:`: A prefix that must appear immediately after the start function declaration token.

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

## Training:

Fron: https://github.com/google-gemini/gemma-cookbook/blob/main/FunctionGemma/%5BFunctionGemma%5DFinetune_FunctionGemma_270M_for_Mobile_Actions_with_Hugging_Face.ipynb

Set up development environment

```python
pip install torch
pip install -U transformers==4.57.1 trl==0.25.1 datasets==4.4.1
```

To fine-tune FunctionGemma, we utilize the Mobile Actions dataset, which is publicly available on Hugging Face. Each entry in this dataset provides:

- The set of tools (functions) the model can use:
- Turn the flashlight on
- Turn the flashlight off
- Create a contact in the phone's contact list
- Send an email

```python
import json
from random import randint
from datasets import load_dataset
from transformers import AutoTokenizer
from huggingface_hub import hf_hub_download

data_file = hf_hub_download(repo_id="google/mobile-actions", filename="dataset.jsonl", repo_type="dataset")
dataset = load_dataset("text", data_files=data_file, encoding="utf-8")["train"].shuffle()

print(f"\n\033[1mHere's an example from your dataset:\033[0m \n{json.dumps(json.loads(dataset[randint(0, len(dataset) - 1)]['text']), indent=2)}")
```

```
Here's an example from your dataset: 
{
  "metadata": "eval",
  "tools": [
    {
      "function": {
        "name": "send_email",
        "description": "Sends an email.",
        "parameters": {
          "type": "OBJECT",
          "properties": {
            "body": {
              "type": "STRING",
              "description": "The body of the email."
            },
            "to": {
              "type": "STRING",
              "description": "The email address of the recipient."
            },
            "subject": {
              "type": "STRING",
              "description": "The subject of the email."
            }
          },
          "required": [
            "to",
            "subject"
          ]
        }
      }
    }
  ],
  "messages": [
    {
      "role": "developer",
      "content": "Current date and time given in YYYY-MM-DDTHH:MM:SS format: 2024-10-20T14:27:48\nDay of week is Sunday\nYou are a model that can do function calling with the following functions\n"
    },
    {
      "role": "user",
      "content": "Please create a calendar event titled \"Q4 Planning Meeting\" for October 24, 2024, at 2:00 PM."
    },
    {
      "role": "assistant",
      "tool_calls": [
        {
          "function": {
            "name": "create_calendar_event",
            "arguments": {
              "title": "Q4 Planning Meeting",
              "datetime": "2024-10-24T14:00:00"
            }
          }
        }
      ]
    }
  ]
}
```

### Process the dataset for training and evaluation

Now that you've loaded your data, format the training dataset into Prompt-completion format for more efficient training later (completion_only_loss=True). This means the model will only learn from the completion instead of the prompt.

- prompt for the non-trainable parts
- completion for the trainable parts

```python
import json

def apply_format(sample):
  template_iputs = json.loads(sample['text'])

  prompt_and_completion = tokenizer.apply_chat_template(
    template_iputs['messages'],
    tools=template_iputs['tools'],
    tokenize=False,
    # add_generation_prompt is False since we don't need model output after all
    # messages.
    add_generation_prompt=False)

  prompt = tokenizer.apply_chat_template(
    template_iputs['messages'][:-1],
    tools=template_iputs['tools'],
    tokenize=False,
    # add_generation_prompt is True since we would like to include
    # "model" in the prompt, if needed.
    add_generation_prompt=True)

  completion = prompt_and_completion[len(prompt):]

  return {
     "prompt": prompt,
     "completion": completion,
     "split": template_iputs["metadata"],
  }

processed_dataset = dataset.map(apply_format)
```

Here's an example from the formatted dataset:

```
  "text": "{\"metadata\": \"train\", \"tools\": [{\"function\": {\"name\": \"send_email\", \"description\": \"Sends an email.\", \"parameters\": {\"type\": \"OBJECT\", \"properties\": {\"subject\": {\"type\": \"STRING\", \"description\": \"The subject of the email.\"}, \"body\": {\"type\": \"STRING\", \"description\": \"The body of the email.\"}, \"to\": {\"type\": \"STRING\", \"description\": \"The email address of the recipient.\"}}, \"required\": [\"to\", \"subject\"]}}}, {\"function\": {\"name\": \"show_map\", \"description\": \"Shows a location on the map.\", \"parameters\": {\"type\": \"OBJECT\", \"properties\": {\"query\": {\"type\": \"STRING\", \"description\": \"The location to search for. May be the name of a place, a business, or an address.\"}}, \"required\": [\"query\"]}}}, {\"function\": {\"name\": \"turn_off_flashlight\", \"description\": \"Turns the flashlight off.\", \"parameters\": {\"type\": \"OBJECT\", \"properties\": {}}}}, {\"function\": {\"name\": \"open_wifi_settings\", \"description\": \"Opens the Wi-Fi settings.\", \"parameters\": {\"type\": \"OBJECT\", \"properties\": {}}}}, {\"function\": {\"name\": \"create_calendar_event\", \"description\": \"Creates a new calendar event.\", \"parameters\": {\"type\": \"OBJECT\", \"properties\": {\"title\": {\"type\": \"STRING\", \"description\": \"The title of the event.\"}, \"datetime\": {\"type\": \"STRING\", \"description\": \"The date and time of the event in the format YYYY-MM-DDTHH:MM:SS.\"}}, \"required\": [\"title\", \"datetime\"]}}}, {\"function\": {\"name\": \"create_contact\", \"description\": \"Creates a contact in the phone's contact list.\", \"parameters\": {\"type\": \"OBJECT\", \"properties\": {\"last_name\": {\"type\": \"STRING\", \"description\": \"The last name of the contact.\"}, \"phone_number\": {\"type\": \"STRING\", \"description\": \"The phone number of the contact.\"}, \"email\": {\"type\": \"STRING\", \"description\": \"The email address of the contact.\"}, \"first_name\": {\"type\": \"STRING\", \"description\": \"The first name of the contact.\"}}, \"required\": [\"first_name\", \"last_name\"]}}}, {\"function\": {\"name\": \"turn_on_flashlight\", \"description\": \"Turns the flashlight on.\", \"parameters\": {\"type\": \"OBJECT\", \"properties\": {}}}}], \"messages\": [{\"role\": \"developer\", \"content\": \"Current date and time given in YYYY-MM-DDTHH:MM:SS format: 2026-01-16T00:50:30\\nDay of week is Friday\\nYou are a model that can do function calling with the following functions\\n\"}, {\"role\": \"user\", \"content\": \"Please schedule a calendar event titled 'Meeting with Mr. Dubois' for January 20th, 2026 at 3:00 PM.\"}, {\"role\": \"assistant\", \"tool_calls\": [{\"function\": {\"name\": \"create_calendar_event\", \"arguments\": {\"title\": \"Meeting with Mr. Dubois\", \"datetime\": \"2026-01-20T15:00:00\"}}}]}]}",
  "prompt": "<bos><start_of_turn>developer\nCurrent date and time given in YYYY-MM-DDTHH:MM:SS format: 2026-01-16T00:50:30\nDay of week is Friday\nYou are a model that can do function calling with the following functions<start_function_declaration>declaration:send_email{description:<escape>Sends an email.<escape>,parameters:{properties:{body:{description:<escape>The body of the email.<escape>,type:<escape>STRING<escape>},subject:{description:<escape>The subject of the email.<escape>,type:<escape>STRING<escape>},to:{description:<escape>The email address of the recipient.<escape>,type:<escape>STRING<escape>}},required:[<escape>to<escape>,<escape>subject<escape>],type:<escape>OBJECT<escape>}}<end_function_declaration><start_function_declaration>declaration:show_map{description:<escape>Shows a location on the map.<escape>,parameters:{properties:{query:{description:<escape>The location to search for. May be the name of a place, a business, or an address.<escape>,type:<escape>STRING<escape>}},required:[<escape>query<escape>],type:<escape>OBJECT<escape>}}<end_function_declaration><start_function_declaration>declaration:turn_off_flashlight{description:<escape>Turns the flashlight off.<escape>,parameters:{type:<escape>OBJECT<escape>}}<end_function_declaration><start_function_declaration>declaration:open_wifi_settings{description:<escape>Opens the Wi-Fi settings.<escape>,parameters:{type:<escape>OBJECT<escape>}}<end_function_declaration><start_function_declaration>declaration:create_calendar_event{description:<escape>Creates a new calendar event.<escape>,parameters:{properties:{datetime:{description:<escape>The date and time of the event in the format YYYY-MM-DDTHH:MM:SS.<escape>,type:<escape>STRING<escape>},title:{description:<escape>The title of the event.<escape>,type:<escape>STRING<escape>}},required:[<escape>title<escape>,<escape>datetime<escape>],type:<escape>OBJECT<escape>}}<end_function_declaration><start_function_declaration>declaration:create_contact{description:<escape>Creates a contact in the phone's contact list.<escape>,parameters:{properties:{email:{description:<escape>The email address of the contact.<escape>,type:<escape>STRING<escape>},first_name:{description:<escape>The first name of the contact.<escape>,type:<escape>STRING<escape>},last_name:{description:<escape>The last name of the contact.<escape>,type:<escape>STRING<escape>},phone_number:{description:<escape>The phone number of the contact.<escape>,type:<escape>STRING<escape>}},required:[<escape>first_name<escape>,<escape>last_name<escape>],type:<escape>OBJECT<escape>}}<end_function_declaration><start_function_declaration>declaration:turn_on_flashlight{description:<escape>Turns the flashlight on.<escape>,parameters:{type:<escape>OBJECT<escape>}}<end_function_declaration><end_of_turn>\n<start_of_turn>user\nPlease schedule a calendar event titled 'Meeting with Mr. Dubois' for January 20th, 2026 at 3:00 PM.<end_of_turn>\n<start_of_turn>model\n",
  "completion": "<start_function_call>call:create_calendar_event{datetime:<escape>2026-01-20T15:00:00<escape>,title:<escape>Meeting with Mr. Dubois<escape>}<end_function_call><start_function_response>",
  "split": "train"
```

Prepare train and eval dataset.

```python
train_dataset = processed_dataset.filter(lambda example: example['split'] == 'train')
eval_dataset = processed_dataset.filter(lambda example: example['split'] == 'eval')
```

### Fine-tune the model

```python
import torch
from transformers import AutoModelForCausalLM
from trl import SFTConfig

output_dir = "/content/mobile-actions-functiongemma"  # Where to save your fine-tuned checkpoints
tokenizer = AutoTokenizer.from_pretrained(gemma_model)

args = SFTConfig(
    output_dir=output_dir,                            # Directory to save adapters
    num_train_epochs=2,                               # Number of training epochs
    per_device_train_batch_size=4,                    # Batch size per device during training
    gradient_accumulation_steps=8,                    # Gradient accumulation during training
    logging_strategy="steps",                         # Log every steps
    eval_strategy="steps",                            # Evaluate loss metrics based on steps
    eval_steps=50,                                    # Evaluate loss metrics every 50 steps
    logging_steps=50,                                 # Log loss metrics every 50 steps
    save_strategy="epoch",                            # Save checkpoint every epoch
    learning_rate=1e-5,                               # Learning rate,
    lr_scheduler_type="cosine",                       # Cosine scheduler is often better for full FT
    max_length=max_token_count,                       # Max sequence length for model and packing of the dataset
    gradient_checkpointing=True,                      # Use gradient checkpointing to save memory
    packing=False,                                    # Groups multiple samples in the dataset into a single sequence
    optim="adamw_torch_fused",                        # Use fused adamw optimizer
    bf16=True,                                        # Use bf16 for mixed precision training
    completion_only_loss=True,                        # Train on completion only to improve quality
    report_to="none"                                  # No reporting.
)

base_model = AutoModelForCausalLM.from_pretrained(
    gemma_model,
    device_map="auto",
    dtype=torch.bfloat16,
    attn_implementation='eager')

base_model.config.pad_token_id = tokenizer.pad_token_id

print("Training configured")
```

### Start training

SFTTrainer tokenizes the datasets and trains the base model using the hyperparameters from the previous step.

The training time varies based on a range of factors, such as the size of your dataset or number of epochs. Using a A100 GPU, this takes about 8 minutes for 1 epoch.

```python
from trl import SFTTrainer

# Train and save the fine-tuned model
trainer = SFTTrainer(
    model=base_model,
    args=args,
    train_dataset=train_dataset,
    eval_dataset=eval_dataset,
)

trainer.train()

trainer.save_model(output_dir)
tokenizer.save_pretrained(output_dir)

print(f"Fine-tuned model saved to {output_dir}")
```

### Conversion to .litertlm for on-device deployment

```
pip uninstall -y tensorflow
pip install ai-edge-torch-nightly --force-reinstall
pip install ai-edge-litert-nightly
```

Build the .litertlm from the fine-tuned model


```python
import os
from ai_edge_torch.generative.examples.gemma3 import gemma3
from ai_edge_torch.generative.utilities import converter
from ai_edge_torch.generative.utilities.export_config import ExportConfig
from ai_edge_torch.generative.layers import kv_cache

# Metadata for FunctionGemma
llm_metadata = r"""start_token: {
    token_ids: {
        ids: [ 2 ]
    }
}
stop_tokens: {
    token_str: ""
}
stop_tokens: {
    token_str: ""
}
llm_model_type: {
    function_gemma: {}
}
"""

checkpoint_dir = "/content/mobile-actions-functiongemma"

litertlm_output_dir = '/content/litertlm'
os.makedirs(litertlm_output_dir, exist_ok=True)

# Create the LLM metadata file
metadata_path = os.path.join(litertlm_output_dir, 'base_llm_metadata.textproto')
with open(metadata_path, 'w') as f:
  f.write(llm_metadata)

# Import the weights and build the PyTorch model
pytorch_model = gemma3.build_model_270m(checkpoint_dir)

# Setup the export configurations and parameters for text generation models.
export_config = ExportConfig()
export_config.kvcache_layout = kv_cache.KV_LAYOUT_TRANSPOSED
export_config.mask_as_input = True

# Convert to LiteRT-LM Format
converter.convert_to_litert(
    pytorch_model,
    output_path=litertlm_output_dir,
    output_name_prefix="mobile-actions",
    prefill_seq_len=256,
    kv_cache_max_len=1024,
    quantize="dynamic_int8",
    export_config=export_config,
    tokenizer_model_path=os.path.join(checkpoint_dir, 'tokenizer.model'),
    base_llm_metadata_path=metadata_path,
    output_format="litertlm",
)
```