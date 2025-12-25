<script lang="ts">
  import { onMount } from 'svelte';
  import { AutoTokenizer, env } from '@xenova/transformers';
  import * as ort from 'onnxruntime-web';

  // Types
  interface InferenceResult {
    [key: string]: ort.Tensor;
  }

  env.allowLocalModels = true;
  env.allowRemoteModels = false;
  env.localModelPath = '/models/';
  
  // Configure ONNX Runtime Environment
  ort.env.wasm.wasmPaths = '/models/';
  ort.env.wasm.numThreads = 1; // Limit threads to avoid OOM/Abort errors on some platforms
  ort.env.wasm.proxy = false; // Set to true if you want to run ORT in a separate worker

  // State
  let textInput = 'Hola, cómo estás?';
  let modelPath = 'model.onnx'; // Default model filename
  let tokenizerName = 'onnx-community/functiongemma-270m-it'; 
  let isLoading = false;
  let statusMessage = '';
  let statusType: 'info' | 'success' | 'error' = 'info';
  let results: InferenceResult | null = null;
  let inferenceTime = 0;
  let useWebGPU = false;
  let forceWASM = false; // Option to bypass WebGPU even if supported
  let backendMessage = '';
  
  let tokenizer: any = null;
  let session: ort.InferenceSession | null = null;

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
      
      // Load tokenizer
      const nameToLoad = String(tokenizerName || 'onnx-community/functiongemma-270m-it');
      
      console.log('--- Tokenizer Initialization ---');
      console.log('Path:', nameToLoad);
      console.log('Base:', env.localModelPath);

      try {
        statusMessage = `Initializing AutoTokenizer...`;
        tokenizer = await AutoTokenizer.from_pretrained(nameToLoad, {
          local_files_only: true,
        });
        console.log('✅ AutoTokenizer loaded successfully');
      } catch (tokenizerError: any) {
        console.error('Tokenizer load failed:', tokenizerError);
        statusMessage = `❌ Tokenizer error: ${tokenizerError.message}. Trying remote...`;
        tokenizer = await AutoTokenizer.from_pretrained('Xenova/gemma-tokenizer');
        console.log('✅ AutoTokenizer loaded from remote');
      }
      
      statusMessage = `Loading ONNX model: ${modelPath}...`;
      
      // Configure ONNX Runtime
      const sessionOptions: ort.InferenceSession.SessionOptions = {
        executionProviders: (useWebGPU && !forceWASM) ? ['webgpu', 'wasm'] : ['wasm'],
        graphOptimizationLevel: 'all',
        executionMode: 'sequential',
      };
      
      console.log('Initializing session with options:', sessionOptions);
      
      // Load ONNX model
      const fullPath = `/models_local/${modelPath}`;
      console.log('--- Model Debug Info ---');
      console.log('Model Path:', fullPath);
      console.log('Execution Providers:', sessionOptions.executionProviders);
      
      statusMessage = `Loading model weights: ${modelPath}...`;
      session = await ort.InferenceSession.create(fullPath, sessionOptions);
      console.log('✅ ONNX Session created successfully');
      
    } catch (error: any) {
      console.error('Initialization error details:', error);
      // Log full error for debugging
      if (error && typeof error === 'object') {
        console.log('Error keys:', Object.keys(error));
        console.log('Error stack:', error.stack);
      }
      
      let errorMessage = error instanceof Error ? error.message : String(error);
      
      if (errorMessage.includes('Aborted') || errorMessage.includes('memory')) {
        errorMessage = 'RuntimeError: Aborted(). This is usually an Out-of-Memory (OOM) error in WASM.\n\n';
        errorMessage += 'Why this happened:\n';
        errorMessage += '- Your model (1.7GB) is too large for the 2GB browser WASM limit.\n';
        errorMessage += '- The "External Data" (.onnx_data) file might be missing for quantized models.\n\n';
        errorMessage += 'Solutions:\n';
        errorMessage += '1. Copy BOTH "model_q4.onnx" AND "model_q4.onnx_data" to static/models_local/.\n';
        errorMessage += '2. Use WebGPU (uncheck "Force WASM") to load large models into VRAM instead of RAM.';
      } else if (errorMessage.includes('failed to fetch') || errorMessage.includes('404')) {
        errorMessage = `File not found: ${modelPath}. Ensure it exists in static/models_local/ and that the filename is correct.`;
      }
      
      statusMessage = `❌ Error loading model: ${errorMessage}`;
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
    
    if (!tokenizer || !session) {
      statusMessage = '⚠️ Model not loaded. Initializing...';
      statusType = 'info';
      await initialize();
      if (!session) return;
    }
    
    try {
      isLoading = true;
      results = null;
      
      // Tokenize
      statusMessage = 'Tokenizing text...';
      statusType = 'info';
      const encoded = await tokenizer(textInput);
      
      // Prepare feeds
      statusMessage = 'Preparing inputs...';
      const feeds: Record<string, ort.Tensor> = {};
      
      for (const inputName of session.inputNames) {
        if (inputName.includes('input_ids') && encoded.input_ids) {
          feeds[inputName] = new ort.Tensor(
            'int64',
            BigInt64Array.from(encoded.input_ids.data, (x: number) => BigInt(x)),
            encoded.input_ids.dims
          );
        } else if (inputName.includes('attention_mask') && encoded.attention_mask) {
          feeds[inputName] = new ort.Tensor(
            'int64',
            BigInt64Array.from(encoded.attention_mask.data, (x: number) => BigInt(x)),
            encoded.attention_mask.dims
          );
        } else if (inputName.includes('position_ids')) {
          // Some models require position_ids
          const seqLen = encoded.input_ids.dims[1];
          const posData = new BigInt64Array(seqLen);
          for (let i = 0; i < seqLen; ++i) posData[i] = BigInt(i);
          feeds[inputName] = new ort.Tensor('int64', posData, [1, seqLen]);
        }
      }
      
      // Run inference
      statusMessage = 'Running inference...';
      const startTime = performance.now();
      console.log('Feeds being sent to model:', Object.keys(feeds));
      const output = await session.run(feeds);
      const endTime = performance.now();
      console.log('✅ Inference complete. Outputs:', Object.keys(output));
      inferenceTime = endTime - startTime;
      
      results = output;
      statusMessage = `✅ Inference completed in ${inferenceTime.toFixed(2)}ms`;
      statusType = 'success';
      
    } catch (error) {
      console.error('Inference error:', error);
      statusMessage = `❌ Error during inference: ${error instanceof Error ? error.message : 'Unknown error'}`;
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
    statusMessage = '';
  }

  onMount(() => {
    checkBackendSupport();
    
    // Reset local model path to simple /models/
    if (typeof window !== 'undefined') {
      env.localModelPath = '/models/';
      env.allowRemoteModels = false;
      env.allowLocalModels = true;
    }
  });
</script>

<div class="container">
  <h1>🚀 ONNX Model Inference</h1>
  <p class="subtitle">Test your ONNX model with WebGPU or WASM backend</p>
  
  {#if backendMessage}
    <div class="backend-info" class:webgpu={useWebGPU}>
      {backendMessage}
    </div>
  {/if}
  
  <div class="config-grid">
    <div class="input-group">
      <label for="modelPath">Model Filename (in static/models_local/):</label>
      <input 
        id="modelPath" 
        type="text"
        bind:value={modelPath}
        placeholder="e.g., model.onnx"
      />
      <small class="hint">Note: For quantized models, ensure the <b>.onnx_data</b> file is also in the folder.</small>
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
      <input type="checkbox" bind:checked={forceWASM} on:change={() => { session = null; statusMessage = 'Configuration changed. Please reload model.'; }} />
      Force WASM backend (Disables WebGPU)
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
            [{Array.from(tensor.data as any).slice(0, 10).join(', ')}{tensor.data.length > 10 ? ', ...' : ''}]
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
  
  .config-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 15px;
    margin-bottom: 20px;
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