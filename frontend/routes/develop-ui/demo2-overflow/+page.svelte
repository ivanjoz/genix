<script lang="ts">
	import Popover2 from '$components/popover2/Popover2.svelte';

	let button1: HTMLElement | null = $state(null);
	let button2: HTMLElement | null = $state(null);
	let show1 = $state(false);
	let show2 = $state(false);
</script>

<div class="p-8">
	<h2 class="text-2xl font-bold mb-4">Overflow Hidden Test</h2>
	<p class="text-gray-600 dark:text-gray-400 mb-8">
		This tests that the popover works even inside containers with <code>overflow: hidden</code>.
		Without proper portal rendering, the popover would be clipped.
	</p>

	<!-- Container with overflow:hidden - would clip normal popovers -->
	<div class="border-2 border-red-500 p-4 mb-8" style="overflow: hidden; height: 200px;">
		<h3 class="text-lg font-semibold mb-2 text-red-600">
			❌ Container with overflow: hidden
		</h3>
		<p class="text-sm text-gray-600 mb-4">
			The popover should still appear correctly and NOT be clipped by this container.
		</p>

		<button
			bind:this={button1}
			onclick={() => show1 = !show1}
			class="px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600"
		>
			Click me (inside overflow:hidden)
		</button>

		<Popover2
			referenceElement={button1}
			open={show1}
			placement="bottom"
		>
			<div class="popover2-container">
				<div class="popover2-content">
					<strong>✅ Success!</strong>
					<br />
					This popover is rendered in the document.body, so it escapes the overflow:hidden parent.
					<br /><br />
					<button
						onclick={() => show1 = false}
						class="px-2 py-1 bg-gray-200 dark:bg-gray-700 rounded text-sm"
					>
						Close
					</button>
				</div>
			</div>
		</Popover2>
	</div>

	<!-- Nested overflow containers -->
	<div class="border-2 border-purple-500 p-4" style="overflow: hidden; height: 250px;">
		<h3 class="text-lg font-semibold mb-2 text-purple-600">
			❌ Nested Container (also overflow: hidden)
		</h3>

		<div class="border-2 border-orange-500 p-4 mt-2" style="overflow: hidden; height: 150px;">
			<h4 class="text-md font-semibold mb-2 text-orange-600">
				❌ Double nested with overflow: hidden
			</h4>
			<p class="text-sm text-gray-600 mb-4">
				Even with multiple nested overflow:hidden containers, the popover works!
			</p>

			<button
				bind:this={button2}
				onclick={() => show2 = !show2}
				class="px-4 py-2 bg-green-500 text-white rounded hover:bg-green-600"
			>
				Deeply nested button
			</button>

			<Popover2
				referenceElement={button2}
				open={show2}
				placement="right"
			>
				<div class="popover2-container">
					<div class="popover2-content">
						<strong>✅ Portal Magic!</strong>
						<br />
						This button is inside 2 levels of overflow:hidden, but the popover still renders perfectly in the body.
						<br /><br />
						<button
							onclick={() => show2 = false}
							class="px-2 py-1 bg-gray-200 dark:bg-gray-700 rounded text-sm"
						>
							Close
					</button>
					</div>
				</div>
			</Popover2>
		</div>
	</div>

	<!-- Normal container for comparison -->
	<div class="border-2 border-green-500 p-4 mt-8">
		<h3 class="text-lg font-semibold mb-2 text-green-600">
			✅ Normal Container (no overflow hidden)
		</h3>
		<p class="text-sm text-gray-600">
			For comparison, this is a normal container without overflow issues.
		</p>
	</div>

	<!-- Instructions -->
	<div class="mt-8 p-4 bg-blue-50 dark:bg-blue-900 rounded">
		<h3 class="font-semibold mb-2">🔍 How to verify:</h3>
		<ol class="list-decimal ml-5 space-y-1 text-sm">
			<li>Click the buttons inside the red and purple containers</li>
			<li>The popovers should appear fully visible, not clipped</li>
			<li>Open the browser DevTools and inspect the popover element</li>
			<li>You should see it's a direct child of <code>&lt;body&gt;</code>, not inside the overflow containers</li>
			<li>Try scrolling and resizing - the popover stays positioned correctly</li>
		</ol>
	</div>
</div>

<style>
	code {
		background: rgba(0, 0, 0, 0.1);
		padding: 2px 6px;
		border-radius: 3px;
		font-size: 0.9em;
		font-family: monospace;
	}
</style>
