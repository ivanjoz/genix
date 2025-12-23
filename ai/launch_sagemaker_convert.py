import sagemaker
from sagemaker.huggingface import HuggingFace
import boto3
import os
import json
import argparse
from datetime import datetime

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--file", type=str, required=True, 
                        help="S3 URI of the model.tar.gz file to convert")
    parser.add_argument("--model_id", type=str, default="google/functiongemma-270m-it",
                        help="Base model ID from HuggingFace")
    parser.add_argument("--instance_type", type=str, default="ml.g4dn.xlarge",
                        help="SageMaker instance type")
    parser.add_argument("--local", action="store_true", help="Run locally using Docker")
    args, _ = parser.parse_known_args()

    # 1. Load configuration
    script_dir = os.path.dirname(os.path.abspath(__file__))
    creds_path = os.path.join(script_dir, "credentials.json")
    if not os.path.exists(creds_path):
        # Try parent directory if not in current
        creds_path = os.path.join(os.path.dirname(script_dir), "credentials.json")
    
    with open(creds_path, "r") as f:
        config = json.load(f)

    aws_profile = config.get("AWS_PROFILE", "default")
    region = config.get("AWS_REGION", "us-east-1")
    role = config.get("SAGEMAKER_ROLE", config.get("SAGEMAKER_IAM_ROLE"))

    # 2. Initialize Session
    if args.local:
        from sagemaker.local import LocalSession
        sess = LocalSession()
        sess.config = {'local': {'local_code': True}}
        instance_type = 'local_gpu' if os.path.exists('/dev/nvidia0') else 'local'
        print(f"🚀 Running in LOCAL MODE ({instance_type})")
    else:
        boto_sess = boto3.Session(profile_name=aws_profile, region_name=region)
        sess = sagemaker.Session(boto_session=boto_sess)
        instance_type = args.instance_type
        print(f"🚀 Running on SageMaker ({instance_type}) [SPOT INSTANCES ENABLED]")

    # 3. Configure Output Path
    s3_output_base = config.get("SAGEMAKER_S3_OUTPUT")
    if s3_output_base:
        if not s3_output_base.endswith("/"):
            s3_output_base += "/"
        bucket = s3_output_base.split("/")[2]
        prefix = "/".join(s3_output_base.split("/")[3:]).rstrip("/")
    else:
        bucket = sess.default_bucket()
        prefix = "gemma-onnx-conversion"

    output_path = f"s3://{bucket}/{prefix}/onnx_conversion"

    print("=" * 70)
    print("FunctionGemma ONNX Conversion - AWS SageMaker")
    print(f"Model to convert: {args.file}")
    print(f"Base Model ID:    {args.model_id}")
    print(f"Output Path:      {output_path}")
    print("=" * 70)

    # 4. Configure HuggingFace Estimator
    huggingface_estimator = HuggingFace(
        entry_point='convert_onnx.py',
        source_dir='./training',
        instance_type=instance_type,
        instance_count=1,
        role=role,
        transformers_version='4.46', # Compatible version
        pytorch_version='2.3',
        py_version='py311',
        hyperparameters={
            "model_id": args.model_id
        },
        output_path=output_path,

        # === SPOT INSTANCES CONFIGURATION ===
        use_spot_instances=not args.local,
        max_wait=3600 if not args.local else None,
        max_run=1800 if not args.local else None,

        environment={
            "HUGGING_FACE_HUB_TOKEN": config.get("HUGGING_FACE_TOKEN", ""),
            "TOKENIZERS_PARALLELISM": "false",
        }
    )

    # 5. Launch the Job
    # We pass the model.tar.gz as the 'model' input channel.
    # SageMaker will download and extract it to /opt/ml/input/data/model/
    inputs = {
        "model": args.file
    }

    print("Launching Conversion Job...")
    huggingface_estimator.fit(inputs=inputs, wait=True)

    print("\n" + "=" * 70)
    print("Conversion Completed!")
    print("=" * 70)
    print(f"Job Name: {huggingface_estimator.latest_training_job.name}")
    print(f"ONNX Model artifacts (tar.gz): {huggingface_estimator.model_data}")
    
    # Information on how to use the output
    print("\nThe ONNX files are inside the model.tar.gz at the S3 path above.")
    print("To download and extract:")
    print(f"  aws s3 cp {huggingface_estimator.model_data} ./onnx_model.tar.gz")
    print(f"  mkdir onnx_model && tar -xzf onnx_model.tar.gz -C onnx_model")

if __name__ == "__main__":
    main()

