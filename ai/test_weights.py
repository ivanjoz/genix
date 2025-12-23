import torch
from transformers import AutoTokenizer, AutoModelForCausalLM
from peft import PeftModel

def test_pytorch():
    base_model_path = "/home/ivanjoz/projects/genix/ai/models/functiongemma-270m-it"
    adapter_path = "/home/ivanjoz/projects/genix/ai/models/functiongemma_270m_lora"

    print("Loading base model in PyTorch (CPU)...")
    base_model = AutoModelForCausalLM.from_pretrained(
        base_model_path,
        torch_dtype=torch.float32,
        trust_remote_code=True,
        device_map="cpu"
    )

    print("Loading LoRA adapter...")
    model = PeftModel.from_pretrained(base_model, adapter_path)
    
    tokenizer = AutoTokenizer.from_pretrained(adapter_path)

    query = "Hola, ¿cómo estás? Cuéntame un chiste."
    
    messages = [
        {"role": "developer", "content": "El modelo es un asistente para ejecutar acciones del sistema Genix en español. Es un ERP para pequeñas empresas que permite gestión de inventarios, ventas, compras, productos, cajas y finanzas. Si no entiendes la acción que el usuario desea realizar, responde exactamente con: no entendí la acción que deseas realizar. You are a model that can do function calling with the following functions"},
        {"role": "user", "content": query}
    ]

    prompt = tokenizer.apply_chat_template(messages, tokenize=False, add_generation_prompt=True)
    inputs = tokenizer(prompt, return_tensors="pt")

    print(f"\nPrompting with: {query}")
    with torch.no_grad():
        outputs = model.generate(**inputs, max_new_tokens=50)
    
    response = tokenizer.decode(outputs[0][len(inputs.input_ids[0]):], skip_special_tokens=True)
    print(f"\nResponse: {response}")

if __name__ == "__main__":
    test_pytorch()

