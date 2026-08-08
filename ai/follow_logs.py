#!/usr/bin/env python3
"""Quick script to follow logs of the most recent training job"""
import sagemaker
import boto3
import tomllib
import os

# Load config
script_dir = os.path.dirname(os.path.abspath(__file__))
parent_dir = os.path.dirname(script_dir)
config_path = os.path.join(parent_dir, "config.toml")
with open(config_path, "rb") as f:
    config = tomllib.load(f)
aws_config = config.get("aws", {})

# AWS setup
boto_sess = boto3.Session(
    profile_name=aws_config.get("profile", "default"),
    region_name=aws_config.get("region", "us-east-1")
)
sess = sagemaker.Session(boto_session=boto_sess)
sm_client = boto_sess.client('sagemaker')

# Get most recent job
response = sm_client.list_training_jobs(
    SortBy='CreationTime',
    SortOrder='Descending',
    MaxResults=1
)

if not response['TrainingJobSummaries']:
    print("No training jobs found.")
    exit(0)

job = response['TrainingJobSummaries'][0]
job_name = job['TrainingJobName']
status = job['TrainingJobStatus']

print(f"\n📊 Job: {job_name}")
print(f"📈 Status: {status}")
print(f"🕐 Created: {job['CreationTime']}")
print("\n" + "="*70)
print("Streaming logs... (Press Ctrl+C to stop)")
print("="*70 + "\n")

try:
    sess.logs_for_job(job_name, wait=True)
except KeyboardInterrupt:
    print("\n\nStopped.")

