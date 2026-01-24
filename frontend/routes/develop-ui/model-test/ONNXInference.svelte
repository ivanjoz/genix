<script lang="ts">
  import { onMount } from 'svelte';
  import { AutoTokenizer, AutoModelForCausalLM, AutoConfig, env, type Tensor, type PreTrainedModel, type PreTrainedTokenizer } from '@huggingface/transformers';

  // Types
  interface DisplayTensor {
    dims: number[];
    type: string;
    data: any[];
    length: number;
  }

  interface InferenceResult {
    [key: string]: DisplayTensor;
  }

  // Configure environment for @huggingface/transformers v3
  env.allowLocalModels = true;
  env.allowRemoteModels = false;
  env.localModelPath = '/models/';
  
  // Configure backends safely
  if (env.backends && env.backends.onnx && env.backends.onnx.wasm) {
    (env.backends.onnx.wasm as any).wasmPaths = '/models/';
    (env.backends.onnx.wasm as any).numThreads = 1;
    (env.backends.onnx.wasm as any).proxy = false;
  }

  // State
  let textInput = 'Hola, cómo estás?';
  let modelPath = 'model_q4.onnx'; // Default to q4 to save RAM
  let tokenizerName = 'onnx-community/functiongemma-270m-it'; 
  let isLoading = false;
  let statusMessage = '';
  let statusType: 'info' | 'success' | 'error' = 'info';
  let results: InferenceResult | null = null;
  let generatedText = '';
  let inferenceTime = 0;
  let useWebGPU = false;
  let forceWASM = false; // Option to bypass WebGPU even if supported
  let backendMessage = '';
  let useSimpleGeneration = false; // Toggle for minimal generation params
  
  let tokenizer: PreTrainedTokenizer | null = null;
  let model: PreTrainedModel | null = null;

  // Check WebGPU support
  async function checkBackendSupport() {
    if (typeof navigator !== 'undefined' && 'gpu' in navigator) {
      try {
        const adapter = await (navigator as any).gpu.requestAdapter();
        if (adapter) {
          useWebGPU = true;
          backendMessage = '✅ WebGPU is supported! Using GPU acceleration.';
          return;
        }
      } catch (e) {
        console.warn('WebGPU check failed:', e);
      }
    }
    
    backendMessage = '⚠️ WebGPU not available. Using WASM (CPU) backend.';
  }

  // Initialize model
  async function initialize() {
    try {
      isLoading = true;
      statusMessage = `Loading tokenizer: ${tokenizerName}...`;
      statusType = 'info';
      
      const nameToLoad = String(tokenizerName || 'onnx-community/functiongemma-270m-it');
      
      console.log('--- Initialization ---');
      console.log('Tokenizer/Model:', nameToLoad);
      console.log('Model File:', modelPath);

      // Set standard v3 environment
      env.localModelPath = '/models/';
      env.allowLocalModels = true;
      env.allowRemoteModels = false;

      // 1. Load tokenizer
      try {
        tokenizer = await AutoTokenizer.from_pretrained(nameToLoad, {
          local_files_only: true,
        });
        console.log('✅ Tokenizer loaded successfully');
      } catch (tokenizerError: any) {
        console.error('Tokenizer load failed:', tokenizerError);
        statusMessage = `❌ Tokenizer error: ${tokenizerError.message}. Trying remote...`;
        env.allowRemoteModels = true;
        tokenizer = await AutoTokenizer.from_pretrained('Xenova/gemma-tokenizer');
        env.allowRemoteModels = false;
        console.log('✅ Tokenizer loaded from remote fallback');
      }
      
      // 2. Load Model using standard v3 structure
      statusMessage = `Loading model: ${modelPath}...`;
      
      const device = (useWebGPU && !forceWASM) ? 'webgpu' : 'wasm';
      console.log('Using device:', device);

      // Since we moved files to static/models/onnx-community/functiongemma-270m-it/onnx/
      // the standard from_pretrained will now find them correctly.
      model = await AutoModelForCausalLM.from_pretrained(nameToLoad, {
        model_file: modelPath, 
        device: device,
        local_files_only: true,
        // v3 specific performance/memory options
        dtype: modelPath.includes('fp16') ? 'fp16' : (modelPath.includes('q4') ? 'q4' : 'fp32'),
      } as any);
      
      console.log('✅ Model loaded successfully');
      statusMessage = '✅ Model and tokenizer ready!';
      statusType = 'success';
      
    } catch (error: any) {
      console.error('Initialization error details:', error);
      
      let errorMessage = error instanceof Error ? error.message : String(error);
      
      if (errorMessage.includes('Aborted') || errorMessage.includes('memory')) {
        errorMessage = 'RuntimeError: Out-of-Memory (OOM) or WASM limit reached.\n\n';
        errorMessage += 'Solutions:\n';
        errorMessage += '1. Ensure you have the .onnx_data file if the model is large.\n';
        errorMessage += '2. Try using WebGPU instead of WASM.';
      } else if (errorMessage.includes('failed to fetch') || errorMessage.includes('404')) {
        errorMessage = `File not found. Check if /static/models_local/${modelPath} and /static/models/${tokenizerName}/config.json exist.`;
      }
      
      statusMessage = `❌ Error: ${errorMessage}`;
      statusType = 'error';
    } finally {
      isLoading = false;
    }
  }

  // Run inference
  async function runInference() {
    if (!textInput.trim()) {
      statusMessage = '⚠️ Please enter some text';
      statusType = 'error';
      return;
    }
    
    if (!tokenizer || !model) {
      statusMessage = '⚠️ Model not loaded. Initializing...';
      statusType = 'info';
      await initialize();
      if (!model) return;
    }
    
    try {
      isLoading = true;
      results = null;
      generatedText = '';
      
      // 1. Prepare formatted prompt for Gemma
      let formattedPrompt = textInput;
      if (!textInput.includes('<start_of_turn>')) {
        // Simple user/model format (no system role - not supported)
        formattedPrompt = `<start_of_turn>user\n${textInput}<end_of_turn>\n<start_of_turn>model\n`;
      }
      
      console.log('Formatted prompt:', formattedPrompt);

      // 2. Tokenize
      statusMessage = 'Tokenizing text...';
      if (!tokenizer) throw new Error('Tokenizer not initialized');
      const encoded = await tokenizer(formattedPrompt);
      console.log('Tokenized input:', encoded);
      
      const input_ids = encoded.input_ids || encoded;
      if (!input_ids) throw new Error('Failed to extract input_ids from tokenizer output');

      // 3. Run inference
      statusMessage = 'Generating response...';
      const startTime = performance.now();
      
      if (!model) throw new Error('Model not initialized');
      if (typeof (model as any).generate !== 'function') {
        throw new Error('Model loaded but generate() method is missing.');
      }
      
      const generationInputs: any = { input_ids };
      if (encoded.attention_mask) {
        generationInputs.attention_mask = encoded.attention_mask;
      }

      // Debug: Log input length
      console.log('Input token count:', input_ids.dims?.[1] || input_ids.data?.length || 0);
      
      // Choose generation strategy
      const genParams = useSimpleGeneration 
        ? {
            max_new_tokens: 10240,
            // Minimal params - let model use defaults
          }
        : {
            max_new_tokens: 10240,
            do_sample: true,
            temperature: 1, // Higher temperature for more variation
            top_k: 50,
            top_p: 0.95,
            repetition_penalty: 1.1,
          };
      
      console.log('Generation params:', genParams);
      const outputTokens = await (model as any).generate(generationInputs, genParams);
      
      const endTime = performance.now();
      inferenceTime = endTime - startTime;
      
      // Debug: Log output token info
      const outputTokenData = outputTokens[0];
      const inputLength = input_ids.dims?.[1] || input_ids.data?.length || 0;
      const outputLength = outputTokenData.data?.length || outputTokenData.size || 
                          (Array.isArray(outputTokenData) ? outputTokenData.length : 0);
      console.log('Input length:', inputLength);
      console.log('Output tokens shape:', outputTokenData.dims || 'unknown');
      console.log('Output tokens total length:', outputLength);
      console.log('New tokens generated:', outputLength - inputLength);
      
      // 4. Decode the full output 
      if (!tokenizer) throw new Error('Tokenizer not initialized');
      
      // First decode everything with special tokens
      const decodedFull = await tokenizer.decode(outputTokens[0], { 
        skip_special_tokens: false 
      });
      
      console.log('=== Full decoded output ===');
      console.log(decodedFull);
      console.log('=== End full output ===');
      
      // Also decode with special tokens skipped
      const decodedClean = await tokenizer.decode(outputTokens[0], {
        skip_special_tokens: true
      });
      console.log('=== Decoded (skip special tokens) ===');
      console.log(decodedClean);
      console.log('=== End cleaned output ===');
      
      // 5. Extract just the model's response
      // Try multiple strategies to extract the response
      
      // Strategy 1: Find text after the last <start_of_turn>model
      let strategy1 = '';
      const modelMarker = '<start_of_turn>model';
      const modelIndex = decodedFull.lastIndexOf(modelMarker);
      if (modelIndex !== -1) {
        strategy1 = decodedFull.substring(modelIndex + modelMarker.length);
        // Remove <end_of_turn> if present
        const endIdx = strategy1.indexOf('<end_of_turn>');
        if (endIdx !== -1) {
          strategy1 = strategy1.substring(0, endIdx);
        }
        strategy1 = strategy1.trim();
        
        // Remove leading "model" word if present (sometimes appears after the tag)
        if (strategy1.toLowerCase().startsWith('model')) {
          strategy1 = strategy1.substring(5).trim();
        }
        // Also check for newline + model pattern
        if (strategy1.startsWith('\n')) {
          strategy1 = strategy1.substring(1).trim();
          if (strategy1.toLowerCase().startsWith('model')) {
            strategy1 = strategy1.substring(5).trim();
          }
        }
      }
      
      // Strategy 2: Use the skip_special_tokens version and remove prompt
      let strategy2 = decodedClean;
      const promptIndex = strategy2.indexOf(textInput);
      if (promptIndex !== -1) {
        strategy2 = strategy2.substring(promptIndex + textInput.length).trim();
      }
      // Also remove "model" prefix if present
      if (strategy2.toLowerCase().startsWith('model')) {
        strategy2 = strategy2.substring(5).trim();
      }
      
      console.log('Strategy 1 (parse markers):', strategy1);
      console.log('Strategy 2 (skip special):', strategy2);
      
      // Use the longer, non-empty result
      generatedText = (strategy1.length > strategy2.length ? strategy1 : strategy2) || strategy1 || strategy2 || 'No output generated';
      
      console.log('=== FINAL Generated Text ===');
      console.log(generatedText);
      console.log('=== Length:', generatedText.length, '===');

      results = null; 
      statusMessage = `✅ Generation completed in ${inferenceTime.toFixed(2)}ms`;
      statusType = 'success';
    } catch (error) {
      console.error('Inference error:', error);
      statusMessage = `❌ Inference error: ${error instanceof Error ? error.message : 'Unknown error'}`;
      statusType = 'error';
    } finally {
      isLoading = false;
    }
  }

  function loadGemmaTemplate() {
    textInput = `<start_of_turn>developer
You are a model that can do function calling with the following functions<start_function_declaration>declaration:get_weather{description:<escape>Obtener el clima de una ciudad<escape>,parameters:{properties:{city:{type:<escape>STRING<escape>}},required:[<escape>city<escape>]}}<end_function_declaration><end_of_turn>
<start_of_turn>user
¿Qué tiempo hace en Madrid?<end_of_turn>
<start_of_turn>model
`;
  }

  function clearResults() {
    results = null;
    generatedText = '';
    statusMessage = '';
  }

  onMount(() => {
    checkBackendSupport();
    
    // Configure environment on mount
    env.allowLocalModels = true;
    env.allowRemoteModels = false;
    env.localModelPath = '/models/';
    if (env.backends && env.backends.onnx && env.backends.onnx.wasm) {
      (env.backends.onnx.wasm as any).wasmPaths = '/models/';
      (env.backends.onnx.wasm as any).numThreads = 1;
      (env.backends.onnx.wasm as any).proxy = false;
    }
  });
</script>

<div class="container">
  <h1>🚀 ONNX Model Inference</h1>
  <p class="subtitle">Test your ONNX model with WebGPU or WASM backend using @huggingface/transformers</p>
  
  {#if backendMessage}
    <div class="backend-info" class:webgpu={useWebGPU}>
      {backendMessage}
    </div>
  {/if}
  
  <div class="config-grid">
    <div class="input-group">
      <label for="modelPath">Model Version:</label>
      <select id="modelPath" bind:value={modelPath} on:change={() => { model = null; statusMessage = 'Model changed. Please run inference to reload.'; }}>
        <option value="model.onnx">Full Precision (FP32)</option>
        <option value="model_fp16.onnx">Half Precision (FP16)</option>
        <option value="model_q4.onnx">Quantized 4-bit (Best for RAM)</option>
        <option value="model_q4f16.onnx">Quantized 4-bit/FP16</option>
      </select>
      <small class="hint">The <b>.onnx_data</b> file is loaded automatically from the same folder.</small>
    </div>

    <div class="input-group">
      <label for="tokenizerName">Tokenizer Name/Path:</label>
      <input 
        id="tokenizerName" 
        type="text"
        bind:value={tokenizerName}
        placeholder="e.g., gemma-tokenizer"
      />
    </div>
  </div>

  <div class="input-group">
    <label for="textInput">Input Text:</label>
    <textarea 
      id="textInput" 
      bind:value={textInput}
      rows="3" 
      placeholder="Enter text for inference..."
    ></textarea>
  </div>

  <div class="config-group">
    <label class="checkbox-container">
      <input type="checkbox" bind:checked={forceWASM} on:change={() => { model = null; statusMessage = 'Configuration changed. Please reload model.'; }} />
      Force WASM backend (Disables WebGPU)
    </label>
    <label class="checkbox-container">
      <input type="checkbox" bind:checked={useSimpleGeneration} />
      Use simple generation (minimal parameters)
    </label>
  </div>
  
  <div class="button-group">
    <button 
      class="run-btn" 
      on:click={runInference}
      disabled={isLoading}
    >
      {#if isLoading}
        <span class="loader"></span>
        Processing...
      {:else}
        Run Inference
      {/if}
    </button>
    <button class="template-btn" on:click={loadGemmaTemplate}>
      Load Gemma Template
    </button>
    <button class="clear-btn" on:click={clearResults}>
      Clear Results
    </button>
  </div>
  
  {#if statusMessage}
    <div class="status {statusType}">
      {statusMessage}
    </div>
  {/if}

  {#if generatedText}
    <div class="generated-text">
      <h3>Generated Response:</h3>
      <div class="text-box">{generatedText}</div>
      <small style="display: block; margin-top: 8px; color: #666;">
        Generation time: {inferenceTime.toFixed(2)}ms
      </small>
    </div>
  {/if}
  
  {#if results}
    <div class="results">
      <h3>Results:</h3>
      
      <div class="result-item">
        <div class="result-label">⏱️ Inference Time:</div>
        <div class="result-value">{inferenceTime.toFixed(2)} ms</div>
      </div>
      
      {#each Object.entries(results) as [name, tensor]}
        <div class="result-item">
          <div class="result-label">📊 {name}:</div>
          <div class="result-value">
            <strong>Shape:</strong> [{tensor.dims.join(', ')}]<br>
            <strong>Type:</strong> {tensor.type}<br>
            <strong>Values (first 10):</strong><br>
            [{tensor.data.slice(0, 10).join(', ')}{tensor.length > 10 ? ', ...' : ''}]
          </div>
        </div>
      {/each}
    </div>
  {/if}
</div>

<style>
  .container {
    background: white;
    border-radius: 20px;
    box-shadow: 0 20px 60px rgba(0,0,0,0.1);
    max-width: 800px;
    width: 100%;
    padding: 40px;
    margin: 40px auto;
  }
  
  h1 {
    color: #333;
    margin-bottom: 10px;
    font-size: 28px;
  }
  
  .subtitle {
    color: #666;
    margin-bottom: 30px;
    font-size: 14px;
  }
  
  .backend-info {
    background: #fff3cd;
    border: 1px solid #ffc107;
    padding: 10px;
    border-radius: 6px;
    margin-bottom: 20px;
    font-size: 13px;
    color: #856404;
  }
  
  .backend-info.webgpu {
    background: #d4edda;
    border-color: #28a745;
    color: #155724;
  }
  
  .warning-box {
    background: #fff3cd;
    border: 1px solid #ffc107;
    padding: 12px;
    border-radius: 6px;
    margin-bottom: 20px;
    font-size: 13px;
    color: #856404;
    line-height: 1.5;
  }
  
  .config-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 15px;
    margin-bottom: 20px;
  }

  select {
    width: 100%;
    padding: 10px;
    border: 2px solid #e0e0e0;
    border-radius: 8px;
    font-size: 14px;
    background: white;
  }

  select:focus {
    outline: none;
    border-color: #667eea;
  }

  .input-group input {
    width: 100%;
    padding: 10px;
    border: 2px solid #e0e0e0;
    border-radius: 8px;
    font-size: 14px;
  }

  .input-group input:focus {
    outline: none;
    border-color: #667eea;
  }

  .hint {
    display: block;
    margin-top: 5px;
    font-size: 11px;
    color: #888;
    line-height: 1.4;
  }

  .input-group {
    margin-bottom: 20px;
  }
  
  .config-group {
    margin-bottom: 20px;
    padding: 10px;
    background: #f0f2f5;
    border-radius: 8px;
  }

  .checkbox-container {
    display: flex;
    align-items: center;
    gap: 10px;
    font-size: 14px;
    cursor: pointer;
    color: #444;
  }

  .checkbox-container input {
    cursor: pointer;
  }
  
  label {
    display: block;
    margin-bottom: 8px;
    color: #555;
    font-weight: 500;
  }
  
  textarea {
    width: 100%;
    padding: 12px;
    border: 2px solid #e0e0e0;
    border-radius: 8px;
    font-size: 16px;
    font-family: inherit;
    resize: vertical;
    transition: border-color 0.3s;
  }
  
  textarea:focus {
    outline: none;
    border-color: #667eea;
  }
  
  .button-group {
    display: flex;
    gap: 10px;
    margin-bottom: 20px;
  }
  
  button {
    flex: 1;
    padding: 14px;
    border: none;
    border-radius: 8px;
    font-size: 16px;
    font-weight: 600;
    cursor: pointer;
    transition: all 0.3s;
  }
  
  .run-btn {
    background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
    color: white;
  }
  
  .run-btn:hover:not(:disabled) {
    transform: translateY(-2px);
    box-shadow: 0 6px 20px rgba(102, 126, 234, 0.4);
  }
  
  .run-btn:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }
  
  .clear-btn {
    background: #f5f5f5;
    color: #666;
  }
  
  .clear-btn:hover {
    background: #e0e0e0;
  }

  .template-btn {
    background: #e3f2fd;
    color: #1976d2;
    border: 1px solid #bbdefb;
  }

  .template-btn:hover {
    background: #bbdefb;
  }
  
  .status {
    padding: 12px;
    border-radius: 8px;
    margin-bottom: 20px;
    font-size: 14px;
  }
  
  .status.info {
    background: #e3f2fd;
    color: #1976d2;
    border-left: 4px solid #1976d2;
  }
  
  .status.success {
    background: #e8f5e9;
    color: #388e3c;
    border-left: 4px solid #388e3c;
  }
  
  .status.error {
    background: #ffebee;
    color: #c62828;
    border-left: 4px solid #c62828;
  }

  .generated-text {
    background: #f3e5f5;
    padding: 20px;
    border-radius: 12px;
    margin-bottom: 20px;
    border: 1px solid #ce93d8;
  }

  .generated-text h3 {
    margin-top: 0;
    font-size: 16px;
    color: #7b1fa2;
  }

  .text-box {
    background: white;
    padding: 15px;
    border-radius: 8px;
    font-family: 'JetBrains Mono', monospace;
    font-size: 18px;
    color: #333;
    border-left: 4px solid #9c27b0;
  }
  
  .results {
    background: #f8f9fa;
    border-radius: 8px;
    padding: 20px;
  }
  
  .results h3 {
    color: #333;
    margin-bottom: 15px;
    font-size: 18px;
  }
  
  .result-item {
    background: white;
    padding: 15px;
    border-radius: 6px;
    margin-bottom: 10px;
    border-left: 3px solid #667eea;
  }
  
  .result-label {
    font-weight: 600;
    color: #555;
    margin-bottom: 5px;
  }
  
  .result-value {
    color: #333;
    font-family: 'Courier New', monospace;
    font-size: 13px;
    word-break: break-all;
  }
  
  .loader {
    display: inline-block;
    width: 16px;
    height: 16px;
    border: 3px solid rgba(255,255,255,.3);
    border-radius: 50%;
    border-top-color: white;
    animation: spin 1s ease-in-out infinite;
    margin-right: 8px;
  }
  
  @keyframes spin {
    to { transform: rotate(360deg); }
  }
</style>