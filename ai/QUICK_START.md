# FunctionGemma Training - Quick Start

## 🚀 Launch Training with Spot Instances

### 1. Configure Credentials
Ensure `../credentials.json` has your `AWS_PROFILE`, `AWS_REGION`, `SAGEMAKER_ROLE`, and optionally `SAGEMAKER_S3_OUTPUT`.

### 2. Run Script
```bash
# 1. Set your Hugging Face token
export HF_TOKEN=your_huggingface_token

# 2. Launch training (uses spot instances automatically)
python launch_sagemaker_train.py
```

## 💰 Cost Savings
- **On-Demand:** ~$0.74/hour
- **Spot Instance:** ~$0.22/hour
- **Your Savings:** ~70% ✨

## 📊 Monitor Progress

```bash
# View in AWS Console
https://console.aws.amazon.com/sagemaker/home#/jobs

# Or use AWS CLI
aws sagemaker list-training-jobs --sort-by CreationTime --sort-order Descending
```

## 📥 Download Model After Training

```bash
# The training output will show the S3 URI
# Download it with:
aws s3 cp s3://your-bucket/path/to/model.tar.gz ./model.tar.gz
tar -xzf model.tar.gz
```

## 🔧 Quick Configuration Changes

### Want faster checkpoints?
Edit `launch_sagemaker_train.py`:
```python
"save_steps": 25,  # Save every 25 steps (default: 50)
```

### Want longer training?
```python
"epochs": 5,  # Train for 5 epochs (default: 3)
```

### Want a bigger instance?
```python
instance_type='ml.g5.xlarge',  # Bigger GPU (NVIDIA A10G)
```

## 📚 Learn More

- **Full Spot Instance Guide:** [SPOT_INSTANCES_GUIDE.md](./SPOT_INSTANCES_GUIDE.md)
- **FunctionGemma Details:** [FUNCTION_GEMMA.md](./FUNCTION_GEMMA.md)
- **Training Script:** [training/train.py](./training/train.py)

## ❓ Common Issues

### "Capacity error"
Wait a few minutes - spot capacity fluctuates. The job will retry automatically.

### "Permission denied"
Check your SageMaker execution role has S3 permissions.

### "Token error"
Make sure `HF_TOKEN` is set: `export HF_TOKEN=hf_...`

---

**Ready?** → `python launch_sagemaker_train.py` 🎯

