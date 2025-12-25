import os
import torch
from ai_edge_torch.generative.examples.gemma3 import gemma3
from ai_edge_torch.generative.utilities import converter
from ai_edge_torch.generative.utilities.export_config import ExportConfig
from ai_edge_torch.generative.layers import kv_cache

# Metadata for FunctionGemma
# As documented in FUNCTION_GEMMA.md (based on Google Colab example)
llm_metadata = r"""start_token: {
    token_ids: {
        ids: [ 2 ]
    }
}
stop_tokens: {
    token_str: "<end_of_turn>"
}
llm_model_type: {
    gemma: {}
}
"""

def convert_model():
    # Base directory for the project
    base_dir = os.path.dirname(os.path.abspath(__file__))
    
    # Paths for input and output
    checkpoint_dir = os.path.join(base_dir, "models/functiongemma_270m_finetuned")
    litertlm_output_dir = os.path.join(base_dir, "models/litertlm")
    
    # Ensure output directory exists
    os.makedirs(litertlm_output_dir, exist_ok=True)

    # Create the LLM metadata file
    metadata_path = os.path.join(litertlm_output_dir, 'base_llm_metadata.textproto')
    print(f"[*] Creating metadata file at: {metadata_path}")
    with open(metadata_path, 'w') as f:
        f.write(llm_metadata)

    # Import the weights and build the PyTorch model
    # gemma3.build_model_270m expects the directory containing the model files (e.g., model.safetensors)
    print(f"[*] Building PyTorch model from: {checkpoint_dir}")
    if not os.path.exists(checkpoint_dir):
        print(f"[!] Error: checkpoint directory {checkpoint_dir} does not exist.")
        return
        
    pytorch_model = gemma3.build_model_270m(checkpoint_dir)

    # Setup the export configurations for text generation models
    export_config = ExportConfig()
    export_config.kvcache_layout = kv_cache.KV_LAYOUT_TRANSPOSED
    export_config.mask_as_input = True

    # Convert to LiteRT-LM Format
    # Note: Added 'web=True' parameter for Web and WebGPU support as requested
    # Added prompt prefix/suffix parameters required by build_litertlm for Gemma 3
    print(f"[*] Converting to LiteRT-LM format in: {litertlm_output_dir}")
    converter.convert_to_litert(
        pytorch_model,
        output_path=litertlm_output_dir,
        output_name_prefix="functiongemma_finetuned",
        prefill_seq_len=256,
        kv_cache_max_len=1024,
        quantize="fp16",
        export_config=export_config,
        tokenizer_model_path=os.path.join(checkpoint_dir, 'tokenizer.model'),
        base_llm_metadata_path=metadata_path,
        output_format="litertlm",
        web=True, # The extra parameter for web and webgpu deployment
        model_prompt_prefix='<start_of_turn>model\n',
        model_prompt_suffix='<end_of_turn>\n',
        user_prompt_prefix='<start_of_turn>user\n',
        user_prompt_suffix='<end_of_turn>\n',
    )
    print("[+] Conversion complete! The model is saved in models/litertlm/")

if __name__ == "__main__":
    convert_model()

