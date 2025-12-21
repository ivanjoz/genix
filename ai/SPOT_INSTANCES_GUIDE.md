# AWS Spot Instances Training Guide for FunctionGemma

## Overview
Your training setup is now optimized to use **AWS Spot Instances**, which can save up to **70% on training costs** compared to on-demand instances.

## What are Spot Instances?
Spot instances are spare EC2 capacity that AWS offers at steep discounts. However, AWS can reclaim them with a 2-minute warning when capacity is needed elsewhere.

## Cost Comparison

| Instance Type | On-Demand Price | Spot Price | Savings |
|---------------|----------------|------------|---------|
| ml.g4dn.xlarge (NVIDIA T4) | ~$0.74/hour | ~$0.22/hour | ~70% |
| ml.g5.xlarge (NVIDIA A10G) | ~$1.41/hour | ~$0.42/hour | ~70% |
| ml.g5.2xlarge | ~$2.03/hour | ~$0.61/hour | ~70% |

## How It Works

### 1. Automatic Checkpointing
The training script now saves checkpoints every **50 steps** to Amazon S3:
```
s3://your-bucket/gemma-function-calling/checkpoints/
```

### 2. Automatic Resumption
If your spot instance is interrupted:
- ✅ The latest checkpoint is automatically saved to S3
- ✅ A new spot instance is launched
- ✅ Training resumes from the last checkpoint
- ✅ No progress is lost!

### 3. Configuration
The system is configured with:
- `max_run=3600` (1 hour max per attempt)
- `max_wait=7200` (2 hours total wait time for spot capacity)
- `checkpoint_s3_uri` - Persistent storage in S3

## Usage

### Basic Launch (with Spot Instances enabled)
```bash
python launch_sagemaker_train.py
```

That's it! The script now automatically uses spot instances.

## Key Features Implemented

### In `launch_sagemaker_train.py`:
✅ `use_spot_instances=True` - Enables spot pricing
✅ `checkpoint_s3_uri` - S3 location for checkpoints
✅ `max_wait` and `max_run` - Timeout management
✅ Updated hyperparameters for FunctionGemma
✅ Cost-tracking tags

### In `training/train.py`:
✅ Automatic checkpoint detection
✅ Resume from latest checkpoint
✅ Frequent checkpoint saving (every 50 steps)
✅ Support for `SM_CHECKPOINT_DIR` environment variable
✅ Optimized for memory efficiency with gradient checkpointing

## Advanced Configuration

### Adjust Checkpoint Frequency
Edit `launch_sagemaker_train.py`:
```python
hyperparameters = {
    "save_steps": 100,  # Save every 100 steps instead of 50
    ...
}
```

### Change Instance Type
For larger models or faster training:
```python
instance_type='ml.g5.xlarge',  # 1x A10G → 24GB VRAM
# or
instance_type='ml.g5.2xlarge',  # 1x A10G → 24GB VRAM (better CPU)
```

### Increase Training Time
For longer training jobs:
```python
max_run=7200,    # 2 hours per attempt
max_wait=14400,  # 4 hours total wait
```

## Monitoring

### View Training Progress
```bash
# From AWS Console
SageMaker → Training Jobs → [your-job-name]

# Or using AWS CLI
aws sagemaker describe-training-job --training-job-name <job-name>
```

### Check Spot Savings
After training completes, check the job details:
```python
# The console will show:
# - Billable seconds (actual compute time)
# - Training time (total elapsed time)
# - Savings from spot instances
```

## Download Trained Model

```bash
# Get the S3 model location from the output
aws s3 cp s3://your-bucket/path/to/model.tar.gz ./model.tar.gz
tar -xzf model.tar.gz

# Or use Python
import boto3
s3 = boto3.client('s3')
# Download from the model_data URI shown after training
```

## Troubleshooting

### Issue: Spot Capacity Not Available
**Solution:** The job will wait up to `max_wait` seconds. If capacity isn't available, consider:
- Using a different region
- Using a different instance type (e.g., ml.g5.xlarge)
- Increasing `max_wait`

### Issue: Training Taking Too Long
**Solution:** Increase `max_run`:
```python
max_run=10800,  # 3 hours
max_wait=21600, # 6 hours
```

### Issue: Checkpoints Not Saving
**Verify:**
1. Check S3 permissions in your IAM role
2. Verify `checkpoint_s3_uri` is accessible
3. Look for errors in CloudWatch logs

## Cost Estimation

For a typical FunctionGemma training job:
- **Dataset size:** 100-500 examples
- **Training time:** ~20-60 minutes
- **Instance:** ml.g4dn.xlarge
- **Cost:**
  - On-demand: $0.74/hour × 1 hour = **$0.74**
  - Spot: $0.22/hour × 1 hour = **$0.22**
  - **Savings: $0.52 (70%)**

## Best Practices

1. ✅ **Test First**: Use `max_steps=50` for quick validation
2. ✅ **Save Frequently**: Keep `save_steps=50` for spot resilience
3. ✅ **Monitor Costs**: Use AWS Cost Explorer with the tags
4. ✅ **Use Appropriate Instance**: ml.g4dn.xlarge is perfect for 270M models
5. ✅ **Check Availability**: Different regions have different spot availability

## Environment Variables (for manual execution)

If running the training script directly (not via SageMaker):
```bash
export SM_CHANNEL_TRAIN=/path/to/data
export SM_MODEL_DIR=/path/to/output
export SM_CHECKPOINT_DIR=/path/to/checkpoints
export HF_TOKEN=your_huggingface_token

python training/train.py \
  --model_id google/functiongemma-270m-it \
  --epochs 3 \
  --batch_size 4 \
  --save_steps 50
```

## Additional Resources

- [AWS Spot Instance Documentation](https://docs.aws.amazon.com/sagemaker/latest/dg/model-managed-spot-training.html)
- [SageMaker Checkpointing](https://docs.aws.amazon.com/sagemaker/latest/dg/model-checkpoints.html)
- [FunctionGemma Documentation](./FUNCTION_GEMMA.md)

---

**Ready to train?** Just run:
```bash
python launch_sagemaker_train.py
```

Your training will automatically use spot instances with full checkpoint management! 🚀

