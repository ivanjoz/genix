<script lang="ts">
import { Popover2 } from '$components/popover2/Popover2.svelte';
	import './popover2.css';
	
	let topButton: HTMLElement | null = $state(null);
	let bottomButton: HTMLElement | null = $state(null);
	let leftButton: HTMLElement | null = $state(null);
	let rightButton: HTMLElement | null = $state(null);
	let autoButton: HTMLElement | null = $state(null);
	
	let showTop = $state(false);
	let showBottom = $state(false);
	let showLeft = $state(false);
	let showRight = $state(false);
	let showAuto = $state(false);
	
	let currentPlacement = $state<string>('bottom');
</script>

<div class="p-8 space-y-8 mt-[50vh]">
	<div>
		<h2 class="text-2xl font-bold mb-2">Popover2 Library</h2>
		<p class="text-gray-600 dark:text-gray-400">
			A simple library that renders elements in the body with smart positioning based on available space.
		</p>
	</div>

	<!-- Fixed Placement Examples -->
	<div class="space-y-4">
		<h3 class="text-xl font-semibold">Fixed Placement (will flip if no space)</h3>
		
		<div class="flex flex-wrap gap-4">
			<!-- Top Placement -->
			<div>
				<button
					bind:this={topButton}
					onclick={() => showTop = !showTop}
					class="px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600"
				>
					Top Placement
				</button>
				<Popover2
					referenceElement={topButton}
					open={showTop}
					placement="top"
				>
					<div class="popover2-container">
						<div class="popover2-content">
							This popover prefers top placement but will flip if there's no space above.
						</div>
					</div>
				</Popover2>
			</div>

			<!-- Bottom Placement -->
			<div>
				<button
					bind:this={bottomButton}
					onclick={() => showBottom = !showBottom}
					class="px-4 py-2 bg-green-500 text-white rounded hover:bg-green-600"
				>
					Bottom Placement
				</button>
				<Popover2
					referenceElement={bottomButton}
					open={showBottom}
					placement="bottom"
				>
					<div class="popover2-container">
						<div class="popover2-content">
							This popover prefers bottom placement but will flip if there's no space below.
						</div>
					</div>
				</Popover2>
			</div>

			<!-- Left Placement -->
			<div>
				<button
					bind:this={leftButton}
					onclick={() => showLeft = !showLeft}
					class="px-4 py-2 bg-purple-500 text-white rounded hover:bg-purple-600"
				>
					Left Placement
				</button>
				<Popover2
					referenceElement={leftButton}
					open={showLeft}
					placement="left"
				>
					<div class="popover2-container">
						<div class="popover2-content">
							This popover prefers left placement.
						</div>
					</div>
				</Popover2>
			</div>

			<!-- Right Placement -->
			<div>
				<button
					bind:this={rightButton}
					onclick={() => showRight = !showRight}
					class="px-4 py-2 bg-orange-500 text-white rounded hover:bg-orange-600"
				>
					Right Placement
				</button>
				<Popover2
					referenceElement={rightButton}
					open={showRight}
					placement="right"
				>
					<div class="popover2-container">
						<div class="popover2-content">
							This popover prefers right placement.
						</div>
					</div>
				</Popover2>
			</div>
		</div>
	</div>

	<!-- Auto Placement Example -->
	<div class="space-y-4">
		<h3 class="text-xl font-semibold">Auto Placement (smart positioning)</h3>
		<p class="text-sm text-gray-600 dark:text-gray-400">
			Current placement: <strong>{currentPlacement}</strong>
		</p>
		
		<div>
			<button
				bind:this={autoButton}
				onclick={() => showAuto = !showAuto}
				class="px-4 py-2 bg-indigo-500 text-white rounded hover:bg-indigo-600"
			>
				Auto Placement
			</button>
			<Popover2
				referenceElement={autoButton}
				open={showAuto}
				placement="bottom"
				onPositionUpdate={(pos) => currentPlacement = pos.placement}
			>
				<div class="popover2-container">
					<div class="popover2-content">
						<strong>Smart Positioning!</strong>
						<br />
						This popover automatically finds the best position based on available space.
						Try scrolling the page or resizing the window.
						<br />
						<br />
						Current: <strong>{currentPlacement}</strong>
					</div>
				</div>
			</Popover2>
		</div>
	</div>

	<!-- Scroll Test Area -->
	<div class="space-y-4">
		<h3 class="text-xl font-semibold">Scroll Test</h3>
		<p class="text-sm text-gray-600 dark:text-gray-400">
			Click the button below, then scroll to see how the popover repositions.
		</p>
	</div>

	<!-- Add spacing to test scroll behavior -->
	<div class="h-[150vh]"></div>

	<div class="sticky bottom-8 left-0 right-0 flex justify-center">
		<button
			class="px-6 py-3 bg-red-500 text-white rounded-lg hover:bg-red-600 shadow-lg"
			onclick={() => showAuto = !showAuto}
		>
			Bottom Button (Test Scroll)
		</button>
	</div>
</div>

<style>
	.space-y-8 > * + * {
		margin-top: 2rem;
	}
	
	.space-y-4 > * + * {
		margin-top: 1rem;
	}
	
	.flex {
		display: flex;
	}
	
	.flex-wrap {
		flex-wrap: wrap;
	}
	
	.gap-4 {
		gap: 1rem;
	}
</style>

