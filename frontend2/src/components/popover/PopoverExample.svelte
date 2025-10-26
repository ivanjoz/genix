<script lang="ts">
	import { Popover } from './index';
	import './popover.css';
</script>

<div class="p-8">
	<h2 class="text-2xl font-bold mb-4">Popover Example</h2>

	<!-- Basic Popover -->
	<Popover.Root>
		<Popover.Trigger class="px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600">
			Click me!
		</Popover.Trigger>

		<Popover.Positioner>
			<Popover.Content class="max-w-xs">
				<Popover.Title>Popover Title</Popover.Title>
				<Popover.Description>
					This is a popover description. You can put any content here!
				</Popover.Description>

				<Popover.CloseTrigger class="ml-auto mt-2">
					✕
				</Popover.CloseTrigger>
			</Popover.Content>
		</Popover.Positioner>
	</Popover.Root>

	<!-- Popover with Arrow -->
	<div class="mt-8">
		<Popover.Root>
			<Popover.Trigger class="px-4 py-2 bg-green-500 text-white rounded hover:bg-green-600">
				With Arrow
			</Popover.Trigger>

			<Popover.Positioner>
				<Popover.Arrow>
					<Popover.ArrowTip />
				</Popover.Arrow>

				<Popover.Content class="max-w-xs">
					<Popover.Title>Popover with Arrow</Popover.Title>
					<Popover.Description>
						This popover includes an arrow pointing to the trigger element.
					</Popover.Description>
				</Popover.Content>
			</Popover.Positioner>
		</Popover.Root>
	</div>
  <div class="mt-[80vh]"></div>
  <Popover.Root>
		<Popover.Trigger class="px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600">
			Click me!
		</Popover.Trigger>

		<Popover.Positioner>
			<Popover.Content class="max-w-xs">
				<Popover.Title>Popover Title</Popover.Title>
				<Popover.Description>
					This is a popover description. You can put any content here!
				</Popover.Description>

				<Popover.CloseTrigger class="ml-auto mt-2">
					✕
				</Popover.CloseTrigger>
			</Popover.Content>
		</Popover.Positioner>
	</Popover.Root>
</div>
