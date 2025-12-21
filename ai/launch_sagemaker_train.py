import sagemaker
from sagemaker.huggingface import HuggingFace
import boto3
import os
import json
from datetime import datetime

# 0. Cargar configuración desde credentials.json (en la carpeta padre)
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

# Inicializar sesión con el perfil y región del JSON
boto_sess = boto3.Session(profile_name=aws_profile, region_name=region)
sess = sagemaker.Session(boto_session=boto_sess)

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
print(f"S3 Path:     {s3_output_base or f's3://{bucket}/{prefix}'}")
print("=" * 70)

# 1. Subir data a S3
local_data_path = "gemma_training_data.jsonl"
s3_data_uri = sess.upload_data(
    path=local_data_path, 
    bucket=bucket, 
    key_prefix=f"{prefix}/data" if prefix else "data"
)
print(f"✓ Data uploaded to: {s3_data_uri}")

# 2. Configurar directorio de checkpoints en S3 para spot instances
checkpoint_s3_uri = f"s3://{bucket}/{prefix}/checkpoints" if prefix else f"s3://{bucket}/checkpoints"
print(f"✓ Checkpoints will be saved to: {checkpoint_s3_uri}")

# 3. Hiperparámetros del modelo (actualizados para FunctionGemma)
hyperparameters = {
    "model_id": "google/functiongemma-270m-it",  # Modelo correcto para function calling
    "epochs": 3,
    "batch_size": 4,
    "gradient_accumulation_steps": 4,  # Batch efectivo = 4 * 4 = 16
    "lr": 2e-4,
    "max_seq_length": 2048,  # FunctionGemma soporta hasta 32K
    "lora_r": 16,
    "lora_alpha": 32,
    "logging_steps": 10,
    "save_steps": 50,  # Guardar checkpoint cada 50 steps (importante para spot)
}

print("\nTraining Configuration:")
for key, value in hyperparameters.items():
    print(f"  {key}: {value}")
print()

# 4. Configurar el estimador de HuggingFace con soporte para Spot Instances
huggingface_estimator = HuggingFace(
    entry_point='train.py',
    source_dir='./training',
    instance_type='ml.g4dn.xlarge',  # NVIDIA T4 - GPU equilibrada para 270M
    instance_count=1,
    role=role,
    transformers_version='4.36',
    pytorch_version='2.1',
    py_version='py310',
    hyperparameters=hyperparameters,
    output_path=f"s3://{bucket}/{prefix}/output" if prefix else f"s3://{bucket}/output",
    
    # === CONFIGURACIÓN PARA SPOT INSTANCES ===
    use_spot_instances=True,           # Activar spot instances (ahorro 70%)
    max_wait=7200,                     # Tiempo máximo de espera (2 horas)
    max_run=3600,                      # Tiempo máximo de ejecución (1 hora)
    checkpoint_s3_uri=checkpoint_s3_uri,  # S3 para guardar checkpoints
    checkpoint_local_path='/opt/ml/checkpoints',  # Path local en el contenedor
    
    # Configuración adicional
    environment={
        "HUGGING_FACE_HUB_TOKEN": os.environ.get("HF_TOKEN", ""),
        "TOKENIZERS_PARALLELISM": "false",  # Evitar warnings
    },
    
    # Etiquetas para organización
    tags=[
        {"Key": "Project", "Value": "FunctionGemma"},
        {"Key": "CostOptimization", "Value": "SpotInstances"},
        {"Key": "Date", "Value": datetime.now().strftime("%Y-%m-%d")}
    ]
)

# 5. Lanzar el Training Job con Spot Instances
print("=" * 70)
print("Launching SageMaker Training Job with Spot Instances...")
print("=" * 70)
print("⚠️  Note: Spot instances can save up to 70% but may be interrupted.")
print("💾 Checkpoints are saved every 50 steps to resume if interrupted.\n")

huggingface_estimator.fit(
    inputs={"train": s3_data_uri},
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

