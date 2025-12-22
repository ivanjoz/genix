#!/usr/bin/env python3
"""Quick script to follow logs of the most recent training job"""
import sagemaker
import boto3
import json
import os

# Load config
script_dir = os.path.dirname(os.path.abspath(__file__))
parent_dir = os.path.dirname(script_dir)
creds_path = os.path.join(parent_dir, "credentials.json")
with open(creds_path, "r") as f:
    config = json.load(f)

# AWS setup
boto_sess = boto3.Session(
    profile_name=config.get("AWS_PROFILE", "default"),
    region_name=config.get("AWS_REGION", "us-east-1")
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

