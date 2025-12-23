import os
import argparse
import torch
from datasets import load_dataset
from transformers import (
    AutoModelForCausalLM,
    AutoTokenizer,
    BitsAndBytesConfig,
    TrainingArguments,
)
from peft import LoraConfig, get_peft_model, prepare_model_for_kbit_training
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

    # Configuración de cuantización para ahorrar memoria (QLoRA)
    # Usar 4-bit para optimizar memoria en GPUs con recursos limitados
    bnb_config = BitsAndBytesConfig(
        load_in_4bit=True,
        bnb_4bit_quant_type="nf4",
        bnb_4bit_compute_dtype=torch.bfloat16,
        bnb_4bit_use_double_quant=True,
    )

    # Obtener el token de HuggingFace del entorno (pasado desde SageMaker)
    hf_token = os.environ.get("HUGGING_FACE_HUB_TOKEN", None)
    if hf_token:
        print("✓ HuggingFace token loaded from environment")
    else:
        print("⚠️  No HuggingFace token found in environment")
    
    # --- LÓGICA DE CACHE EN S3 (DEBE OCURRIR ANTES DE CARGAR EL MODELO) ---
    # 1. Determinar la ruta del modelo a usar
    model_path = None
    
    # Caso 1: Si el modelo ya está disponible en S3 (pasado como input channel)
    if args.model_dir and os.path.exists(args.model_dir):
        model_path = args.model_dir
        print(f"✅ Using cached model from S3: {model_path}")
    
    # Caso 2: Si no está en S3, descargar de HuggingFace y cachear
    else:
        print(f"ℹ️ Model not found in S3, downloading from HuggingFace: {args.model_id}")
        
        # Si queremos cachear en S3, descargar primero a un directorio temporal
        if args.s3_model_cache:
            try:
                from huggingface_hub import snapshot_download
                print(f"📥 Downloading model from HuggingFace to cache...")
                temp_cache_dir = "/tmp/model_cache"
                
                # Descargar el modelo UNA SOLA VEZ
                snapshot_download(
                    repo_id=args.model_id, 
                    local_dir=temp_cache_dir, 
                    token=hf_token
                )
                
                # Subir a S3 usando el CLI de AWS (ya configurado con el rol de SageMaker)
                print(f"⬆️ Uploading model to S3 cache: {args.s3_model_cache}")
                os.system(f"aws s3 sync {temp_cache_dir} {args.s3_model_cache}")
                print("✅ Model cached successfully in S3")
                
                # Usar el modelo descargado localmente
                model_path = temp_cache_dir
                
            except Exception as e:
                print(f"⚠️ Failed to cache model to S3: {e}")
                print(f"ℹ️ Falling back to direct HuggingFace download")
                model_path = args.model_id
        else:
            # Si no hay S3 cache configurado, descargar directamente de HuggingFace
            model_path = args.model_id
            print("ℹ️ No S3 cache configured, loading directly from HuggingFace")
    
    # -----------------------------
    
    # Cargar modelo y processor (CRÍTICO: usar AutoProcessor, no AutoTokenizer)
    # El AutoProcessor de FunctionGemma maneja los control tokens especiales
    print(f"Loading model: {args.model_id}")
    print(f"Model path: {model_path}")
    
    model = AutoModelForCausalLM.from_pretrained(
        model_path,
        quantization_config=bnb_config,
        device_map="auto",
        trust_remote_code=True,
        attn_implementation="eager",  # Use eager attention to avoid warnings
        token=hf_token,
    )
    
    # Cargar tokenizer de FunctionGemma
    # El AutoTokenizer maneja automáticamente los control tokens especiales
    print(f"Loading tokenizer: {args.model_id}")
    tokenizer = AutoTokenizer.from_pretrained(
        model_path, 
        trust_remote_code=True, 
        token=hf_token
    )
    tokenizer.pad_token = tokenizer.eos_token
    tokenizer.padding_side = "right"
    
    # Validar que los control tokens de FunctionGemma estén presentes
    control_tokens = [
        "<start_function_declaration>", 
        "<end_function_declaration>",
        "<start_function_call>", 
        "<end_function_call>",
        "<escape>"
    ]
    print("Validating FunctionGemma control tokens...")
    for token in control_tokens:
        if token not in tokenizer.get_vocab():
            print(f"WARNING: Control token '{token}' not found in tokenizer vocabulary!")
    print("Control tokens validation complete.")

    # Configuración de LoRA optimizada para FunctionGemma
    # Los target_modules cubren todos los proyectores de atención y MLP
    peft_config = LoraConfig(
        r=args.lora_r,
        lora_alpha=args.lora_alpha,
        target_modules=["q_proj", "o_proj", "k_proj", "v_proj", "gate_proj", "up_proj", "down_proj"],
        lora_dropout=0.05,
        bias="none",
        task_type="CAUSAL_LM",
    )

    print("Preparing model for k-bit training...")
    model = prepare_model_for_kbit_training(model)
    model = get_peft_model(model, peft_config)
    
    # Imprimir el número de parámetros entrenables
    trainable_params = sum(p.numel() for p in model.parameters() if p.requires_grad)
    all_params = sum(p.numel() for p in model.parameters())
    print(f"Trainable params: {trainable_params:,} || All params: {all_params:,} || Trainable%: {100 * trainable_params / all_params:.2f}%")

    # Argumentos de entrenamiento optimizados para FunctionGemma y Spot Instances
    training_args = TrainingArguments(
        output_dir=args.output_dir,
        per_device_train_batch_size=args.batch_size,
        gradient_accumulation_steps=args.gradient_accumulation_steps,
        learning_rate=args.lr,
        logging_steps=args.logging_steps,
        num_train_epochs=args.epochs,
        max_steps=args.max_steps if args.max_steps > 0 else -1,
        bf16=True,  # Recomendado para GPUs modernas (A10G, V100, A100)
        
        # Estrategia de guardado optimizada para Spot Instances
        save_strategy="steps" if args.save_steps > 0 else "epoch",
        save_steps=args.save_steps if args.save_steps > 0 else None,
        save_total_limit=3,  # Mantener solo los 3 últimos checkpoints (ahorra espacio)
        save_safetensors=True,  # Use safetensors format (safer and avoids torch.load issues)
        
        # Optimizador y scheduler
        eval_strategy="no",
        optim="paged_adamw_32bit",  # Optimizado para QLoRA
        lr_scheduler_type="cosine",  # Mejor convergencia
        warmup_ratio=0.03,  # Warmup del 3%
        
        # Logging y checkpoints
        report_to="tensorboard",
        logging_dir=os.path.join(args.output_dir, "logs"),
        
        # Optimizaciones de memoria
        gradient_checkpointing=True,  # Ahorrar memoria GPU
        dataloader_num_workers=2,
        
        # IMPORTANTE para Spot Instances: guardar checkpoints en directorio persistente
        # SageMaker sincroniza automáticamente este directorio con S3
        save_on_each_node=True,
        load_best_model_at_end=False,
    )
    
    # Si hay un directorio de checkpoints, asegurarse de que los argumentos apunten allí
    if args.checkpoint_dir and os.path.exists(args.checkpoint_dir):
        # Usar el directorio de checkpoints de SageMaker si está disponible
        # SageMaker sincroniza automáticamente /opt/ml/checkpoints con S3
        training_args.output_dir = args.checkpoint_dir
        print(f"📂 Checkpoints will be saved to: {args.checkpoint_dir}")

    # Definir la función de formateo que usa el tokenizer
    def formatting_func_with_tokenizer(example):
        return formatting_prompts_func(example, tokenizer)

    # Configurar el tokenizer con la longitud máxima de secuencia
    tokenizer.model_max_length = args.max_seq_length
    
    # Configurar el data collator para entrenar solo en la respuesta del asistente
    # En FunctionGemma, el turno del modelo comienza con <start_of_turn>model
    response_template = "<start_of_turn>model\n"
    collator = DataCollatorForCompletionOnlyLM(
        response_template=response_template, 
        tokenizer=tokenizer
    )

    # Trainer SFT (Supervised Fine-Tuning) para FunctionGemma
    print("Initializing SFTTrainer...")
    trainer = SFTTrainer(
        model=model,
        train_dataset=dataset,
        peft_config=peft_config,
        tokenizer=tokenizer,  # Use tokenizer instead of processing_class for compatibility
        data_collator=collator,
        args=training_args,
        formatting_func=formatting_func_with_tokenizer,
        max_seq_length=args.max_seq_length,
    )

    print("Starting training...")
    print(f"Total training steps: {len(dataset) * args.epochs // (args.batch_size * args.gradient_accumulation_steps)}")
    
    # Buscar checkpoints existentes (para reanudar en caso de interrupción de spot instance)
    checkpoint_dir = None
    if os.path.exists(args.checkpoint_dir):
        checkpoints = [os.path.join(args.checkpoint_dir, d) for d in os.listdir(args.checkpoint_dir) 
                      if d.startswith("checkpoint-") and os.path.isdir(os.path.join(args.checkpoint_dir, d))]
        if checkpoints:
            # Ordenar por número de step y tomar el último
            checkpoints.sort(key=lambda x: int(x.split("-")[-1]))
            checkpoint_dir = checkpoints[-1]
            print(f"📂 Found checkpoint: {checkpoint_dir}")
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

    # Guardar el modelo final (adapter LoRA)
    print(f"💾 Saving final model to {args.output_dir}")
    trainer.save_model(args.output_dir)
    
    # Guardar también el tokenizer para facilitar la inferencia
    tokenizer.save_pretrained(args.output_dir)
    print("✅ Training completed successfully!")

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Fine-tune FunctionGemma with QLoRA")
    
    # Model and data arguments
    parser.add_argument(
        "--model_id", 
        type=str, 
        default="google/functiongemma-270m-it",
        help="Model ID from HuggingFace Hub. Default: google/functiongemma-270m-it"
    )
    parser.add_argument(
        "--train_dir", 
        type=str, 
        default=os.environ.get("SM_CHANNEL_TRAIN", "./"),
        help="Directory containing training data (gemma_training_data.jsonl)"
    )
    parser.add_argument(
        "--model_dir", 
        type=str, 
        default=os.environ.get("SM_CHANNEL_MODEL", None),
        help="Directory containing the base model (if cached in S3)"
    )
    parser.add_argument(
        "--s3_model_cache", 
        type=str, 
        default=os.environ.get("S3_MODEL_CACHE", None),
        help="S3 URI to cache the model if downloaded from HF"
    )
    parser.add_argument(
        "--output_dir", 
        type=str, 
        default=os.environ.get("SM_MODEL_DIR", "./output"),
        help="Directory to save the trained model"
    )
    parser.add_argument(
        "--checkpoint_dir",
        type=str,
        default=os.environ.get("SM_CHECKPOINT_DIR", "/opt/ml/checkpoints"),
        help="Directory for checkpoints (for spot instance resumption)"
    )
    
    # Training hyperparameters
    parser.add_argument(
        "--batch_size", 
        type=int, 
        default=4,
        help="Per-device training batch size"
    )
    parser.add_argument(
        "--gradient_accumulation_steps", 
        type=int, 
        default=4,
        help="Number of gradient accumulation steps (effective batch = batch_size * this)"
    )
    parser.add_argument(
        "--epochs", 
        type=int, 
        default=20,
        help="Number of training epochs"
    )
    parser.add_argument(
        "--max_steps", 
        type=int, 
        default=-1,
        help="Maximum number of training steps (overrides epochs if > 0)"
    )
    parser.add_argument(
        "--lr", 
        type=float, 
        default=2e-4,
        help="Learning rate"
    )
    parser.add_argument(
        "--max_seq_length", 
        type=int, 
        default=2048,
        help="Maximum sequence length (FunctionGemma supports up to 32K)"
    )
    
    # LoRA configuration
    parser.add_argument(
        "--lora_r", 
        type=int, 
        default=16,
        help="LoRA attention dimension (rank)"
    )
    parser.add_argument(
        "--lora_alpha", 
        type=int, 
        default=32,
        help="LoRA alpha parameter (scaling factor)"
    )
    
    # Logging and saving
    parser.add_argument(
        "--logging_steps", 
        type=int, 
        default=10,
        help="Log metrics every N steps"
    )
    parser.add_argument(
        "--save_steps", 
        type=int, 
        default=0,
        help="Save checkpoint every N steps (0 = save only at epoch end)"
    )
    
    args = parser.parse_args()
    
    # Validate arguments
    if not os.path.exists(args.train_dir):
        print(f"ERROR: Training directory does not exist: {args.train_dir}")
        exit(1)
    
    train_file = os.path.join(args.train_dir, "gemma_training_data.jsonl")
    if not os.path.exists(train_file):
        print(f"ERROR: Training file not found: {train_file}")
        exit(1)
    
    # Create output directory if it doesn't exist
    os.makedirs(args.output_dir, exist_ok=True)
    
    print("=" * 60)
    print("FunctionGemma Fine-Tuning Configuration")
    print("=" * 60)
    print(f"Model: {args.model_id}")
    print(f"Training data: {train_file}")
    print(f"Output directory: {args.output_dir}")
    print(f"Batch size: {args.batch_size}")
    print(f"Gradient accumulation: {args.gradient_accumulation_steps}")
    print(f"Effective batch size: {args.batch_size * args.gradient_accumulation_steps}")
    print(f"Learning rate: {args.lr}")
    print(f"Epochs: {args.epochs}")
    print(f"Max sequence length: {args.max_seq_length}")
    print(f"LoRA rank: {args.lora_r}")
    print(f"LoRA alpha: {args.lora_alpha}")
    print("=" * 60)
    
    train(args)

