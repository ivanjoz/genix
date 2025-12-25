<script lang="ts">
	import { onMount } from 'svelte';
	import { FilesetResolver, LlmInference } from '@mediapipe/tasks-genai';

	// --- RUNES ($state) ---
	let llm = $state.raw<LlmInference | null>(null);
	let prompt = $state('');
	let responseText = $state('System ready. Waiting for model initialization...');
	let isLoading = $state(false);
	let isModelLoaded = $state(false);

	// Update the path to match your file in /static
	const MODEL_PATH = '/models/functiongemma_finetuned_fp16_ekv1024.bin';

	// --- RUNES ($effect) ---
	// In Svelte 5, $effect runs in the browser. 
	// We use it here to initialize the model once on mount.
	$effect(() => {
		async function initModel() {
			try {
				const genai = await FilesetResolver.forGenAiTasks(
					'https://cdn.jsdelivr.net/npm/@mediapipe/tasks-genai/wasm'
				);
				
				llm = await LlmInference.createFromOptions(genai, {
					baseOptions: { modelAssetPath: MODEL_PATH },
					maxTokens: 1024,
					temperature: 0.4,
					topK: 40
				});

				isModelLoaded = true;
				responseText = 'FunctionGemma is ready!';
			} catch (e) {
				console.error("Model load error:", e);
				responseText = `Error: ${e instanceof Error ? e.message : 'Failed to load model'}`;
			}
		}

		initModel();

		return () => {
			if (llm) {
				llm.close();
			}
		};
	});

	async function generate() {
		if (!prompt || !llm || isLoading){
			console.log("promp:",prompt,llm,isLoading)
			return
		};

		isLoading = true;
		responseText = ''; // Clear for new response

		try {
			// Streaming implementation (better UX)
			await llm.generateResponse(prompt, (partialText, done) => {
				responseText += partialText;
			});
		} catch (e) {
			responseText = "Inference failed. check console.";
			console.error(e);
		} finally {
			isLoading = false;
		}
	}
</script>

<div class="container">
	<h1>Gemma 3 Web Explorer</h1>
	
	<div class="badge" class:loaded={isModelLoaded}>
		{isModelLoaded ? '● Local Model Active' : '○ Downloading Model...'}
	</div>

	<div class="chat-interface">
		<textarea 
			bind:value={prompt} 
			placeholder="Ask your fine-tuned model anything..."
			disabled={!isModelLoaded || isLoading}
		></textarea>

		<button onclick={generate} disabled={!isModelLoaded || isLoading}>
			{isLoading ? 'Thinking...' : 'Run Inference'}
		</button>
	</div>

	<div class="output-panel">
		<strong>Output:</strong>
		<p class="response">{responseText}</p>
	</div>
</div>

<style>
	:global(body) { background: #121212; color: white; font-family: sans-serif; }
	.container { max-width: 800px; margin: 40px auto; padding: 20px; }
	.badge { font-size: 0.8rem; margin-bottom: 20px; color: #aaa; transition: color 0.3s; }
	.loaded { color: #4caf50; font-weight: bold; }
	
	textarea { 
		width: 100%; height: 120px; 
		background: #1e1e1e; color: white; 
		border: 1px solid #333; border-radius: 8px; 
		padding: 12px; font-size: 1rem;
	}
	
	button { 
		margin-top: 10px; padding: 12px 24px; 
		background: #3d5afe; color: white; border: none; 
		border-radius: 6px; cursor: pointer; font-weight: bold;
	}

	button:disabled { opacity: 0.5; cursor: not-allowed; }

	.output-panel { 
		margin-top: 30px; padding: 20px; 
		background: #1e1e1e; border-radius: 12px; border-left: 4px solid #3d5afe; 
	}
	
	.response { white-space: pre-wrap; line-height: 1.6; color: #e0e0e0; }
</style>