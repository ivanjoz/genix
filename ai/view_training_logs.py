#!/usr/bin/env python3
"""
Script to view logs from a running or completed SageMaker training job.
Usage: python view_training_logs.py [job_name]
If no job_name is provided, it will show the most recent job.
"""

import sagemaker
import boto3
import json
import os
import sys

# Load configuration
script_dir = os.path.dirname(os.path.abspath(__file__))
parent_dir = os.path.dirname(script_dir)
creds_path = os.path.join(parent_dir, "credentials.json")
with open(creds_path, "r") as f:
    config = json.load(f)

# AWS configuration
aws_profile = config.get("AWS_PROFILE", "default")
region = config.get("AWS_REGION", "us-east-1")

# Initialize session
boto_sess = boto3.Session(profile_name=aws_profile, region_name=region)
sess = sagemaker.Session(boto_session=boto_sess)
sm_client = boto_sess.client('sagemaker')

def list_recent_jobs(limit=10):
    """List recent training jobs"""
    response = sm_client.list_training_jobs(
        SortBy='CreationTime',
        SortOrder='Descending',
        MaxResults=limit
    )
    
    print("\n" + "=" * 70)
    print("Recent Training Jobs:")
    print("=" * 70)
    for i, job in enumerate(response['TrainingJobSummaries'], 1):
        status = job['TrainingJobStatus']
        name = job['TrainingJobName']
        created = job['CreationTime'].strftime('%Y-%m-%d %H:%M:%S')
        print(f"{i}. {name}")
        print(f"   Status: {status} | Created: {created}")
    print("=" * 70 + "\n")
    
    return [job['TrainingJobName'] for job in response['TrainingJobSummaries']]

def get_job_status(job_name):
    """Get training job status"""
    try:
        response = sm_client.describe_training_job(TrainingJobName=job_name)
        status = response['TrainingJobStatus']
        
        print(f"\nJob: {job_name}")
        print(f"Status: {status}")
        
        if status == 'InProgress':
            if 'SecondaryStatusTransitions' in response and response['SecondaryStatusTransitions']:
                secondary = response['SecondaryStatusTransitions'][-1]['Status']
                print(f"Secondary Status: {secondary}")
        
        if 'TrainingStartTime' in response:
            print(f"Started: {response['TrainingStartTime']}")
        
        if 'TrainingEndTime' in response:
            print(f"Ended: {response['TrainingEndTime']}")
            
        if 'FailureReason' in response:
            print(f"⚠️  Failure Reason: {response['FailureReason']}")
        
        print()
        return status
    except Exception as e:
        print(f"Error getting job status: {e}")
        return None

def stream_logs(job_name):
    """Stream logs from a training job"""
    print(f"Streaming logs for: {job_name}")
    print("=" * 70)
    print("(Press Ctrl+C to stop)\n")
    
    try:
        # Attach to the training job and stream logs
        from sagemaker.estimator import Estimator
        estimator = Estimator.attach(job_name, sagemaker_session=sess)
        
        # This will stream the logs
        sess.logs_for_job(job_name, wait=True)
        
    except KeyboardInterrupt:
        print("\n\nLog streaming stopped by user.")
    except Exception as e:
        print(f"Error streaming logs: {e}")
        print("\nTip: If the job is very recent, logs might not be available yet.")
        print("     Wait a minute and try again.")

if __name__ == "__main__":
    print("\n🔍 SageMaker Training Job Log Viewer")
    
    if len(sys.argv) > 1:
        # Job name provided as argument
        job_name = sys.argv[1]
        get_job_status(job_name)
        stream_logs(job_name)
    else:
        # List recent jobs and let user choose
        recent_jobs = list_recent_jobs()
        
        if not recent_jobs:
            print("No training jobs found.")
            sys.exit(0)
        
        print("Enter job number to view logs (or press Enter for most recent): ", end='')
        choice = input().strip()
        
        if choice == '':
            job_name = recent_jobs[0]
        else:
            try:
                idx = int(choice) - 1
                job_name = recent_jobs[idx]
            except (ValueError, IndexError):
                print("Invalid choice. Using most recent job.")
                job_name = recent_jobs[0]
        
        print()
        get_job_status(job_name)
        stream_logs(job_name)

