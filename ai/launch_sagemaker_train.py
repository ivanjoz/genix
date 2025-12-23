import sagemaker
from sagemaker.huggingface import HuggingFace
import boto3
import os
import json
import argparse
from datetime import datetime
from huggingface_hub import snapshot_download

# 0. Analizar argumentos de línea de comandos
parser = argparse.ArgumentParser()
parser.add_argument("--local", action="store_true", help="Ejecutar localmente usando Docker")
parser.add_argument("--cache_model", action="store_true", default=True, help="Descargar y cachear el modelo en S3 (habilitado por defecto)")
parser.add_argument("--no-cache", action="store_true", help="Descargar el modelo directamente desde HuggingFace sin cachear en S3")
parser.add_argument("--full", action="store_true", help="Realizar Full Fine-Tuning en lugar de LoRA")
args, _ = parser.parse_known_args()

# Si se especifica --no-cache, desactivar el cache
if args.no_cache:
    args.cache_model = False

# 0.1 Cargar configuración desde credentials.json (en la carpeta padre)
script_dir = os.path.dirname(os.path.abspath(__file__))
parent_dir = os.path.dirname(script_dir)
creds_path = os.path.join(parent_dir, "credentials.json")
with open(creds_path, "r") as f:
    config = json.load(f)

# Configuración de AWS
aws_profile = config.get("AWS_PROFILE", "default")
region = config.get("AWS_REGION", "us-east-1")
# El rol de ejecución de SageMaker debe tener permisos de SageMaker y S3
role = config.get("SAGEMAKER_ROLE", config.get("SAGEMAKER_IAM_ROLE"))

# Inicializar sesión
if args.local:
    # Para modo local, necesitamos una sesión especial
    from sagemaker.local import LocalSession
    sess = LocalSession()
    sess.config = {'local': {'local_code': True}}
    instance_type = 'local_gpu' if os.path.exists('/dev/nvidia0') else 'local'
    print(f"🚀 Ejecutando en MODO LOCAL ({instance_type})")
else:
    boto_sess = boto3.Session(profile_name=aws_profile, region_name=region)
    sess = sagemaker.Session(boto_session=boto_sess)
    instance_type = 'ml.g4dn.xlarge'
    print(f"🚀 Ejecutando en SageMaker ({instance_type})")

# Configuración de S3 (usar SAGEMAKER_S3_OUTPUT si existe, si no usar bucket default)
s3_output_base = config.get("SAGEMAKER_S3_OUTPUT")
if s3_output_base:
    # Asegurar que termina en / para concatenar subcarpetas
    if not s3_output_base.endswith("/"):
        s3_output_base += "/"
    bucket = s3_output_base.split("/")[2]
    prefix = "/".join(s3_output_base.split("/")[3:]).rstrip("/")
else:
    bucket = sess.default_bucket()
    prefix = "gemma-function-calling"

print("=" * 70)
print("FunctionGemma Training - AWS SageMaker with Spot Instances")
print(f"AWS Profile: {aws_profile}")
print(f"Region:      {region}")
print(f"Role ARN:    {role}")
print(f"S3 Path:     {s3_output_base or (f's3://{bucket}/{prefix}' if prefix else f's3://{bucket}')}")
print("=" * 70)

# 1. Gestionar Cache del Modelo en S3
model_id = "google/functiongemma-270m-it"
# Construir la URI evitando doble slash cuando prefix está vacío
prefix_path = f"{prefix}/" if prefix else ""
s3_model_uri = f"s3://{bucket}/{prefix_path}model_cache/{model_id.split('/')[-1]}"

# Verificar si el modelo ya está en S3 para evitar descargas/subidas innecesarias
s3_exists = False
if not args.local:
    try:
        s3_client = boto_sess.client('s3')
        # Verificar un archivo clave para confirmar que el modelo está completo
        s3_key = f"{prefix_path}model_cache/{model_id.split('/')[-1]}/config.json"
        s3_client.head_object(Bucket=bucket, Key=s3_key)
        s3_exists = True
        print(f"✅ Modelo encontrado en S3 cache: {s3_model_uri}")
    except:
        print(f"ℹ️ El modelo no está en S3 cache o no es accesible.")

if args.cache_model and not s3_exists:
    print(f"📦 Gestionando cache local para: {model_id}")
    local_cache_dir = os.path.join(script_dir, "cache", model_id.split("/")[-1])
    
    # Descargar modelo si no existe localmente
    if not os.path.exists(local_cache_dir):
        print(f"📥 Descargando modelo desde HuggingFace a {local_cache_dir}...")
        snapshot_download(
            repo_id=model_id,
            local_dir=local_cache_dir,
            token=config.get("HUGGING_FACE_TOKEN", "")
        )
    
    # Subir a S3 si no es modo local
    if not args.local:
        print(f"⬆️ Subiendo modelo a S3: {s3_model_uri}...")
        profile_flag = f" --profile {aws_profile}" if aws_profile and aws_profile != "default" else ""
        os.system(f"aws s3 sync {local_cache_dir} {s3_model_uri}{profile_flag}")
    else:
        # En modo local, usamos el path directo
        s3_model_uri = f"file://{local_cache_dir}"
elif args.local:
    # Si es local y no forzamos cache_model, ver si existe localmente
    local_cache_dir = os.path.join(script_dir, "cache", model_id.split("/")[-1])
    if os.path.exists(local_cache_dir):
        s3_model_uri = f"file://{local_cache_dir}"
        print(f"✅ Usando cache local: {s3_model_uri}")
    else:
        s3_model_uri = None # Dejar que el contenedor descargue de HF
elif not s3_exists:
    # No está en S3 y no pedimos cache_model
    s3_model_uri = None
    print("ℹ️ No se usará cache de S3 (el contenedor descargará de HuggingFace)")

# 2. Subir data a S3
local_data_path = "gemma_training_data.jsonl"
if args.local:
    s3_data_uri = f"file://{os.path.abspath(local_data_path)}"
else:
    s3_data_uri = sess.upload_data(
        path=local_data_path, 
        bucket=bucket, 
        key_prefix=f"{prefix}/data" if prefix else "data"
    )
print(f"✓ Data location: {s3_data_uri}")

# 3. Configurar directorio de checkpoints
if args.local:
    checkpoint_s3_uri = f"file://{os.path.abspath('./checkpoints')}"
else:
    checkpoint_s3_uri = f"s3://{bucket}/{prefix_path}checkpoints"
print(f"✓ Checkpoints will be saved to: {checkpoint_s3_uri}")

# 3. Hiperparámetros del modelo (actualizados para FunctionGemma)
hyperparameters = {
    "model_id": "google/functiongemma-270m-it",  # Modelo correcto para function calling
    "epochs": 10,
    "batch_size": 2,
    "gradient_accumulation_steps": 4,  # Batch efectivo = 4 * 4 = 16
    "lr": 2e-5 if args.full else 2e-4, # LR más bajo para full finetuning
    "max_seq_length": 2048,  # FunctionGemma soporta hasta 32K
    "logging_steps": 10,
    "save_steps": 50,  # Guardar checkpoint cada 50 steps (importante para spot)
}

if not args.full:
    hyperparameters.update({
        "lora_r": 16,
        "lora_alpha": 32,
    })

print("\nTraining Configuration:")
for key, value in hyperparameters.items():
    print(f"  {key}: {value}")
print(f"  Full Finetuning: {args.full}")
print()

# 4. Configurar el estimador de HuggingFace
huggingface_estimator = HuggingFace(
    entry_point='train_full.py' if args.full else 'train.py',
    source_dir='./training',
    instance_type=instance_type,
    instance_count=1,
    role=role,
    transformers_version='4.46',
    pytorch_version='2.3',
    py_version='py311',
    hyperparameters=hyperparameters,
    output_path=f"s3://{bucket}/{prefix}/output" if prefix else f"s3://{bucket}/output",
    
    # === CONFIGURACIÓN PARA SPOT INSTANCES ===
    use_spot_instances=not args.local,   # No usar spot en modo local
    max_wait=7200 if not args.local else None,
    max_run=3600 if not args.local else None,
    checkpoint_s3_uri=checkpoint_s3_uri,
    checkpoint_local_path='/opt/ml/checkpoints',
    
    # Configuración adicional
    environment={
        "HUGGING_FACE_HUB_TOKEN": config.get("HUGGING_FACE_TOKEN", ""),
        "TOKENIZERS_PARALLELISM": "false",  # Evitar warnings
        "S3_MODEL_CACHE": s3_model_uri if not s3_exists else "", # Pasar URI para cachear si no existe
    },
    
    # Etiquetas para organización
    tags=[
        {"Key": "Project", "Value": "FunctionGemma"},
        {"Key": "CostOptimization", "Value": "SpotInstances"},
        {"Key": "Date", "Value": datetime.now().strftime("%Y-%m-%d")}
    ]
)

# 5. Lanzar el Training Job
print("=" * 70)
if args.local:
    print("Launching Local Training Job...")
else:
    print("Launching SageMaker Training Job with Spot Instances...")
print("=" * 70)

# Definir inputs
inputs = {"train": s3_data_uri}
if s3_model_uri:
    inputs["model"] = s3_model_uri

huggingface_estimator.fit(
    inputs=inputs,
    wait=True,
    logs='All'
)

print("\n" + "=" * 70)
print("Training Completed!")
print("=" * 70)
print(f"Job Name: {huggingface_estimator.latest_training_job.name}")
print(f"Model artifacts: {huggingface_estimator.model_data}")
print(f"Checkpoints: {checkpoint_s3_uri}")
print("\nTo download the trained model:")
print(f"  aws s3 cp {huggingface_estimator.model_data} ./model.tar.gz")
print(f"  tar -xzf model.tar.gz")

