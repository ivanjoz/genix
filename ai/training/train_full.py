import os
import argparse
import torch
from datasets import load_dataset
from transformers import (
    AutoModelForCausalLM,
    AutoTokenizer,
    TrainingArguments,
)
try:
    from trl import SFTTrainer, DataCollatorForCompletionOnlyLM
except ImportError:
    # Fallback for older trl versions or different installation structures
    print("⚠️  Standard TRL import failed, trying fallback...")
    from trl.trainer import SFTTrainer
    from trl.trainer.utils import DataCollatorForCompletionOnlyLM

from tools_config import ALL_TOOLS

def formatting_prompts_func(example, tokenizer):
    """
    Formatea un ejemplo individual usando el AutoTokenizer de FunctionGemma.
    Se pasan TODOS los tools para que el modelo aprenda a elegir entre ellos.
    """
    text = tokenizer.apply_chat_template(
        example['messages'],
        tools=ALL_TOOLS,
        tokenize=False,
        add_generation_prompt=False
    )
    return text

def train(args):
    # Cargar dataset generado (JSONL)
    print(f"Loading dataset from: {os.path.join(args.train_dir, 'gemma_training_data.jsonl')}")
    dataset = load_dataset("json", data_files=os.path.join(args.train_dir, "gemma_training_data.jsonl"), split="train")
    print(f"Dataset loaded: {len(dataset)} examples")

    # Obtener el token de HuggingFace del entorno (pasado desde SageMaker)
    hf_token = os.environ.get("HUGGING_FACE_HUB_TOKEN", None)
    if hf_token:
        print("✓ HuggingFace token loaded from environment")
    else:
        print("⚠️  No HuggingFace token found in environment")
    
    # --- LÓGICA DE CACHE EN S3 ---
    model_path = None
    
    # Caso 1: Si el modelo ya está disponible en S3 (pasado como input channel)
    if args.model_dir and os.path.exists(args.model_dir):
        model_path = args.model_dir
        print(f"✅ Using cached model from S3: {model_path}")
    
    # Caso 2: Si no está en S3, descargar de HuggingFace y cachear
    else:
        print(f"ℹ️ Model not found in S3, downloading from HuggingFace: {args.model_id}")
        
        if args.s3_model_cache:
            try:
                from huggingface_hub import snapshot_download
                print(f"📥 Downloading model from HuggingFace to cache...")
                temp_cache_dir = "/tmp/model_cache"
                
                snapshot_download(
                    repo_id=args.model_id, 
                    local_dir=temp_cache_dir, 
                    token=hf_token
                )
                
                print(f"⬆️ Uploading model to S3 cache: {args.s3_model_cache}")
                os.system(f"aws s3 sync {temp_cache_dir} {args.s3_model_cache}")
                print("✅ Model cached successfully in S3")
                model_path = temp_cache_dir
                
            except Exception as e:
                print(f"⚠️ Failed to cache model to S3: {e}")
                model_path = args.model_id
        else:
            model_path = args.model_id
            print("ℹ️ No S3 cache configured, loading directly from HuggingFace")
    
    # --- CARGA DE MODELO PARA FULL FINETUNING ---
    print(f"Loading model for FULL FINETUNING: {args.model_id}")
    
    # Para Full Finetuning no usamos BitsAndBytesConfig
    model = AutoModelForCausalLM.from_pretrained(
        model_path,
        device_map="auto",
        trust_remote_code=True,
        attn_implementation="eager",
        token=hf_token,
        torch_dtype=torch.bfloat16 if torch.cuda.is_bf16_supported() else torch.float16,
    )
    
    # Cargar tokenizer
    print(f"Loading tokenizer: {args.model_id}")
    tokenizer = AutoTokenizer.from_pretrained(
        model_path, 
        trust_remote_code=True, 
        token=hf_token
    )
    tokenizer.pad_token = tokenizer.eos_token
    tokenizer.padding_side = "right"
    
    # Validar que los control tokens de FunctionGemma estén presentes
    control_tokens = ["<start_function_declaration>", "<end_function_declaration>", "<start_function_call>", "<end_function_call>", "<escape>"]
    for token in control_tokens:
        if token not in tokenizer.get_vocab():
            print(f"WARNING: Control token '{token}' not found!")

    # Imprimir el número de parámetros (todos son entrenables en full finetuning)
    all_params = sum(p.numel() for p in model.parameters())
    print(f"Full Finetuning - All params: {all_params:,} (all are trainable)")

    # Argumentos de entrenamiento
    training_args = TrainingArguments(
        output_dir=args.output_dir,
        per_device_train_batch_size=args.batch_size,
        gradient_accumulation_steps=args.gradient_accumulation_steps,
        learning_rate=args.lr,
        logging_steps=args.logging_steps,
        num_train_epochs=args.epochs,
        max_steps=args.max_steps if args.max_steps > 0 else -1,
        bf16=torch.cuda.is_bf16_supported(),
        fp16=not torch.cuda.is_bf16_supported(),
        
        save_strategy="steps" if args.save_steps > 0 else "epoch",
        save_steps=args.save_steps if args.save_steps > 0 else None,
        save_total_limit=3,
        save_safetensors=True,
        
        eval_strategy="no",
        optim="adamw_torch",  # Usar optimizador estándar para full finetuning
        lr_scheduler_type="cosine",
        warmup_ratio=0.03,
        
        report_to="tensorboard",
        logging_dir=os.path.join(args.output_dir, "logs"),
        
        gradient_checkpointing=True,
        dataloader_num_workers=2,
        save_on_each_node=True,
    )
    
    if args.checkpoint_dir and os.path.exists(args.checkpoint_dir):
        training_args.output_dir = args.checkpoint_dir
        print(f"📂 Checkpoints will be saved to: {args.checkpoint_dir}")

    # Definir la función de formateo
    def formatting_func_with_tokenizer(example):
        return formatting_prompts_func(example, tokenizer)

    tokenizer.model_max_length = args.max_seq_length
    
    response_template = "<start_of_turn>model\n"
    collator = DataCollatorForCompletionOnlyLM(
        response_template=response_template, 
        tokenizer=tokenizer
    )

    # Trainer SFT
    print("Initializing SFTTrainer for Full Finetuning...")
    trainer = SFTTrainer(
        model=model,
        train_dataset=dataset,
        tokenizer=tokenizer, # Use tokenizer instead of processing_class for compatibility
        data_collator=collator,
        args=training_args,
        formatting_func=formatting_func_with_tokenizer,
        max_seq_length=args.max_seq_length,
    )

    print("Starting training...")
    
    checkpoint_dir = None
    if os.path.exists(args.checkpoint_dir):
        checkpoints = [os.path.join(args.checkpoint_dir, d) for d in os.listdir(args.checkpoint_dir) 
                      if d.startswith("checkpoint-") and os.path.isdir(os.path.join(args.checkpoint_dir, d))]
        if checkpoints:
            checkpoints.sort(key=lambda x: int(x.split("-")[-1]))
            potential_checkpoint = checkpoints[-1]
            # Verify if it is a valid full finetuning checkpoint
            has_index = any(os.path.exists(os.path.join(potential_checkpoint, f)) 
                          for f in ["model.safetensors.index.json", "pytorch_model.bin.index.json", "model.safetensors", "pytorch_model.bin"])
            if has_index:
                checkpoint_dir = potential_checkpoint
                print(f"📂 Found valid checkpoint: {checkpoint_dir}")
            else:
                print(f"⚠️ Found checkpoint at {potential_checkpoint} but it is incompatible with full finetuning. Ignoring.")
        else:
            print("📂 No previous checkpoints found. Starting from scratch.")
    else:
        print("📂 No checkpoint directory found. Starting from scratch.")

    # Entrenar (reanuda automáticamente si se proporciona checkpoint_dir)
    # Try to resume from checkpoint, but if it fails due to version issues, start fresh
    try:
        trainer.train(resume_from_checkpoint=checkpoint_dir)
    except (ValueError, RuntimeError) as e:
        if "torch.load" in str(e) or "CVE-2025-32434" in str(e) or "safetensors" in str(e):
            print(f"⚠️  Failed to load checkpoint due to version incompatibility: {e}")
            print("🔄 Starting training from scratch...")
            trainer.train()
        else:
            raise

    # Guardar el modelo final completo
    print(f"💾 Saving final full model to {args.output_dir}")
    trainer.save_model(args.output_dir)
    tokenizer.save_pretrained(args.output_dir)
    print("✅ Full Finetuning completed successfully!")

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Full fine-tune FunctionGemma")
    
    parser.add_argument("--model_id", type=str, default="google/functiongemma-270m-it")
    parser.add_argument("--train_dir", type=str, default=os.environ.get("SM_CHANNEL_TRAIN", "./"))
    parser.add_argument("--model_dir", type=str, default=os.environ.get("SM_CHANNEL_MODEL", None))
    parser.add_argument("--s3_model_cache", type=str, default=os.environ.get("S3_MODEL_CACHE", None))
    parser.add_argument("--output_dir", type=str, default=os.environ.get("SM_MODEL_DIR", "./output"))
    parser.add_argument("--checkpoint_dir", type=str, default=os.environ.get("SM_CHECKPOINT_DIR", "/opt/ml/checkpoints"))
    
    parser.add_argument("--batch_size", type=int, default=4)
    parser.add_argument("--gradient_accumulation_steps", type=int, default=4)
    parser.add_argument("--epochs", type=int, default=20)
    parser.add_argument("--max_steps", type=int, default=-1)
    parser.add_argument("--lr", type=float, default=2e-5) # Lower LR for full finetuning usually
    parser.add_argument("--max_seq_length", type=int, default=2048)
    
    parser.add_argument("--logging_steps", type=int, default=10)
    parser.add_argument("--save_steps", type=int, default=0)
    
    args = parser.parse_args()
    
    if not os.path.exists(args.train_dir):
        exit(1)
    
    os.makedirs(args.output_dir, exist_ok=True)
    
    print("=" * 60)
    print("FunctionGemma FULL Fine-Tuning Configuration")
    print("=" * 60)
    print(f"Model: {args.model_id}")
    print(f"Output directory: {args.output_dir}")
    print(f"Batch size: {args.batch_size}")
    print(f"Learning rate: {args.lr}")
    print(f"Epochs: {args.epochs}")
    print("=" * 60)
    
    train(args)

