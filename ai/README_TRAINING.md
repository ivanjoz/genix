# FunctionGemma Training with AWS Spot Instances

## 🚀 Quick Start

### 1. Configure AWS Credentials
The system reads configuration from a `credentials.json` file located in the parent directory (`../credentials.json`).

Ensure your `credentials.json` includes:
```json
{
    "AWS_PROFILE": "your-profile-name",
    "AWS_REGION": "us-east-1",
    "SAGEMAKER_ROLE": "arn:aws:iam::123456789012:role/service-role/AmazonSageMaker-ExecutionRole-...",
    "SAGEMAKER_S3_OUTPUT": "s3://your-bucket/optional-prefix"
}
```
*Note: If `SAGEMAKER_S3_OUTPUT` is not provided, it will use the SageMaker default session bucket.*
*Note: If `SAGEMAKER_ROLE` is not present, it will fallback to `LAMBDA_IAM_ROLE` if available.*

### 2. Install Dependencies
You can install the necessary local dependencies (SageMaker SDK and Boto3) using the provided script:

```bash
# Make the script executable and run it
chmod +x install_launcher.sh
./install_launcher.sh

# Activate the virtual environment
source venv/bin/activate
```

### 3. Launch Training
```bash
# 1. Set your Hugging Face token
export HF_TOKEN=your_huggingface_token

# 2. Launch training
python launch_sagemaker_train.py
```

**That's it!** Your training will run on AWS spot instances with automatic checkpoint management.

## 💰 Cost Savings

Your setup now uses **AWS Spot Instances**:
- **Save up to 70%** on training costs
- **Automatic interruption handling** - no progress lost
- **Checkpoint every 50 steps** to S3

| Training Duration | On-Demand | Spot | You Save |
|-------------------|-----------|------|----------|
| 1 hour | $0.74 | $0.22 | **$0.52 (70%)** |
| 3 hours | $2.22 | $0.66 | **$1.56 (70%)** |

## 📁 Project Structure

```
ai/
├── launch_sagemaker_train.py    # Launch script (spot instances enabled)
├── training/
│   ├── train.py                 # Training script (checkpoint support)
│   └── requirements.txt         # Python dependencies
├── gemma_training_data.jsonl    # Training data
│
├── FUNCTION_GEMMA.md           # FunctionGemma documentation
├── SPOT_INSTANCES_GUIDE.md     # Detailed spot instance guide
├── QUICK_START.md              # Quick reference
├── CHANGES_SUMMARY.md          # What changed
└── README_TRAINING.md          # This file
```

## ✨ Key Features

### ✅ Spot Instance Support
- 70% cost savings
- Automatic checkpoint to S3
- Seamless resumption on interruption

### ✅ FunctionGemma Optimized
- Uses `AutoProcessor` (critical for control tokens)
- Proper handling of `<start_function_call>`, `<escape>`, etc.
- Correct model: `google/functiongemma-270m-it`

### ✅ Production Ready
- QLoRA for memory efficiency
- Gradient checkpointing
- Configurable hyperparameters
- Comprehensive logging

## 🎯 What's Configured

### Instance
- **Type**: `ml.g4dn.xlarge` (NVIDIA T4, 16GB VRAM)
- **Cost**: ~$0.22/hour (spot) vs $0.74/hour (on-demand)
- **Perfect for**: 270M parameter models

### Training
- **Model**: `google/functiongemma-270m-it`
- **Method**: QLoRA (4-bit quantization + LoRA adapters)
- **Batch Size**: 4 (effective: 16 with gradient accumulation)
- **Epochs**: 3 (default)
- **Checkpoints**: Every 50 steps → S3

### LoRA Configuration
- **Rank (r)**: 16
- **Alpha**: 32
- **Target Modules**: All attention + MLP projectors
- **Trainable Params**: ~2-3% of total (efficient!)

## 📊 Monitoring

### AWS Console
```
https://console.aws.amazon.com/sagemaker/home#/jobs
```

### AWS CLI
```bash
# List recent jobs
aws sagemaker list-training-jobs --sort-by CreationTime

# Check specific job
aws sagemaker describe-training-job --training-job-name <name>

# View logs
aws logs tail /aws/sagemaker/TrainingJobs/<job-name> --follow
```

## 🔧 Customization

### More Epochs
Edit `launch_sagemaker_train.py`:
```python
hyperparameters = {
    "epochs": 5,  # Train longer
    ...
}
```

### Faster Checkpoints (for testing)
```python
hyperparameters = {
    "save_steps": 25,  # Save every 25 steps
    ...
}
```

### Bigger Instance
```python
instance_type='ml.g5.xlarge',  # More powerful GPU (NVIDIA A10G)
```

### Longer Training Window
```python
max_run=7200,    # 2 hours per attempt
max_wait=14400,  # 4 hours total
```

## 📥 After Training

### Download Model
```bash
# From training output, copy the S3 URI
aws s3 cp s3://your-bucket/path/to/model.tar.gz ./model.tar.gz
tar -xzf model.tar.gz
```

### Use for Inference
```python
from transformers import AutoProcessor, AutoModelForCausalLM
from peft import PeftModel

# Load base model
base_model = AutoModelForCausalLM.from_pretrained(
    "google/functiongemma-270m-it",
    device_map="auto"
)

# Load LoRA adapter
model = PeftModel.from_pretrained(base_model, "./path/to/adapter")

# Load processor
processor = AutoProcessor.from_pretrained("./path/to/adapter")

# Use for inference
messages = [
    {"role": "developer", "content": "You are a model that can do function calling..."},
    {"role": "user", "content": "¿Qué tiempo hace en Madrid?"}
]

inputs = processor.apply_chat_template(
    messages,
    tools=[weather_tool],
    return_tensors="pt"
).to(model.device)

outputs = model.generate(**inputs, max_new_tokens=128)
print(processor.decode(outputs[0]))
```

## 🆘 Troubleshooting

### Issue: "No spot capacity"
**Solution**: Job will auto-retry. Be patient or try different region/instance.

### Issue: "Permission denied" on S3
**Solution**: Check SageMaker execution role has S3 full access.

### Issue: "Model not found"
**Solution**: Ensure `HF_TOKEN` is set and valid for accessing Gemma models.

### Issue: Training very slow
**Solution**: 
- Check if `bf16=True` is enabled (faster on modern GPUs)
- Increase `batch_size` if memory allows
- Consider larger instance type

## 📖 Documentation

| Document | Purpose |
|----------|---------|
| [QUICK_START.md](./QUICK_START.md) | Fast reference for common tasks |
| [SPOT_INSTANCES_GUIDE.md](./SPOT_INSTANCES_GUIDE.md) | Deep dive into spot instances |
| [FUNCTION_GEMMA.md](./FUNCTION_GEMMA.md) | FunctionGemma architecture |
| [CHANGES_SUMMARY.md](./CHANGES_SUMMARY.md) | What was changed and why |

## 🎓 Best Practices

1. ✅ **Test First**: Run with `max_steps=50` to validate setup
2. ✅ **Monitor Costs**: Use AWS Cost Explorer with tags
3. ✅ **Keep Checkpoints**: Don't delete S3 checkpoint directory during training
4. ✅ **Use Spot**: For production - savings compound over time
5. ✅ **Validate Data**: Ensure JSONL format matches FunctionGemma spec

## 💡 Pro Tips

- **Data Quality > Quantity**: 100 good examples > 1000 mediocre ones
- **Test Locally First**: Use smaller model or CPU for data validation
- **Monitor GPU Usage**: Check if you need bigger/smaller instance
- **Experiment with LoRA r**: Higher rank = more capacity but slower
- **Save Training Logs**: Useful for debugging and optimization

## 🔬 Advanced: Local Testing

For development/testing without SageMaker:

```bash
# Create local environment
python -m venv venv
source venv/bin/activate
pip install -r training/requirements.txt

# Set up paths
export SM_CHANNEL_TRAIN="./"
export SM_MODEL_DIR="./output"
export SM_CHECKPOINT_DIR="./checkpoints"
export HF_TOKEN="your_token"

# Run training
python training/train.py \
  --model_id google/functiongemma-270m-it \
  --epochs 1 \
  --max_steps 10 \
  --batch_size 2
```

## ❓ FAQ

**Q: How much does training cost?**
A: ~$0.22/hour on spot instances. Typical job: 30-60 min = $0.11-$0.22

**Q: What if spot instance is interrupted?**
A: Training resumes automatically from last checkpoint (every 50 steps)

**Q: Can I use on-demand instead?**
A: Yes, set `use_spot_instances=False` in launch script

**Q: How do I know training is working?**
A: Check SageMaker console or CloudWatch logs for progress

**Q: Can I train on multiple GPUs?**
A: Yes, set `instance_count=2` or use multi-GPU instance type

## 🎉 Summary

Your training setup is **production-ready** with:
- ✅ 70% cost savings via spot instances
- ✅ Automatic checkpoint management
- ✅ FunctionGemma-optimized processing
- ✅ Memory-efficient QLoRA training
- ✅ Comprehensive monitoring & logging

**Ready to train?**
```bash
export HF_TOKEN=your_token
python launch_sagemaker_train.py
```

---

**Need help?** Check the [SPOT_INSTANCES_GUIDE.md](./SPOT_INSTANCES_GUIDE.md) or open an issue.

