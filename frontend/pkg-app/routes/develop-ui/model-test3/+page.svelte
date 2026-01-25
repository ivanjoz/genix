<script lang="ts">
	import { onMount } from 'svelte';
	import * as webLLM from "@mlc-ai/web-llm";

	// --- RUNES ($state) ---
	let engine = $state.raw<webLLM.MLCEngineInterface | null>(null);
	let prompt = $state('');
	let responseText = $state('System ready. Waiting for model initialization...');
	let isLoading = $state(false);
	let isModelLoaded = $state(false);
	let progressText = $state('');

	const MODEL_ID = "functiongemma_270m_finetuned";
	const MODEL_URL = "/models/functiongemma_270m_finetuned/";

	// In Svelte 5, $effect runs in the browser. 
	$effect(() => {
		async function initModel() {
			try {
				const fullModelUrl = new URL(MODEL_URL, window.location.origin).href;
				const appConfig: webLLM.AppConfig = {
					model_list: [
						{
							model: fullModelUrl,
							model_id: MODEL_ID,
							model_lib: "https://raw.githubusercontent.com/mlc-ai/binary-mlc-llm-libs/main/web-llm-models/v0_2_80/gemma-3-1b-it-q4f16_1-ctx4k_cs1k-webgpu.wasm",
						}
					]
				};

				engine = await webLLM.CreateMLCEngine(MODEL_ID, {
					appConfig,
					initProgressCallback: (report) => {
						progressText = report.text;
						responseText = report.text;
					}
				});

				isModelLoaded = true;
				responseText = 'FunctionGemma (MLC) is ready!';
			} catch (e) {
				console.error("Model load error:", e);
				responseText = `Error: ${e instanceof Error ? e.message : 'Failed to load model'}`;
			}
		}

		initModel();

		return () => {
			if (engine) {
				engine.unload();
			}
		};
	});

	async function generate() {
		if (!prompt || !engine || isLoading) return;

		isLoading = true;
		responseText = ''; 

		try {
			const messages: webLLM.ChatCompletionMessageParam[] = [
				{ role: "user", content: prompt }
			];

			const chunks = await engine.chat.completions.create({
				messages,
				stream: true,
			});

			for await (const chunk of chunks) {
				const content = chunk.choices[0]?.delta?.content;
				if (content) {
					responseText += content;
				}
			}
		} catch (e) {
			responseText = "Inference failed. Check console.";
			console.error(e);
		} finally {
			isLoading = false;
		}
	}
</script>

<div class="container">
	<h1>Gemma 3 Web Explorer (MLC)</h1>
	
	<div class="badge" class:loaded={isModelLoaded}>
		{isModelLoaded ? '● Local MLC Model Active' : '○ Initializing...'}
	</div>

	{#if !isModelLoaded}
		<div class="progress-info">
			{progressText}
		</div>
	{/if}

	<div class="chat-interface">
		<textarea 
			bind:value={prompt} 
			placeholder="Ask your fine-tuned MLC model anything..."
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
	
	.progress-info {
		font-size: 0.9rem;
		color: #888;
		margin-bottom: 15px;
		font-style: italic;
	}

	textarea { 
		width: 100%; height: 120px; 
		background: #1e1e1e; color: white; 
		border: 1px solid #333; border-radius: 8px; 
		padding: 12px; font-size: 1rem;
		resize: vertical;
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
