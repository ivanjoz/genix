<script lang="ts">
	import Page from "../../../components/Page.svelte";
</script>

<Page>
	<div>hola</div>
</Page>

<style>
	.page-container {
		min-height: 100vh;
		background-color: #f8f9fa;
	}

	.header-section {
		margin-bottom: 1.5rem;
	}

	h2 {
		font-size: 1.5rem;
		font-weight: 600;
		margin: 0;
		color: #1a202c;
	}

	.text-sm {
		font-size: 0.875rem;
	}

	.text-gray-600 {
		color: #6c757d;
	}

	.text-blue-600 {
		color: #2563eb;
	}

	.text-gray-500 {
		color: #6c757d;
	}

	.mt-1 {
		margin-top: 0.25rem;
	}

	.mt-2 {
		margin-top: 0.5rem;
	}

	.selected-info {
		padding: 0.75rem 1rem;
		background-color: #e7f3ff;
		border-left: 4px solid #3b82f6;
		border-radius: 4px;
		color: #1e40af;
		display: flex;
		align-items: center;
		gap: 0.5rem;
	}

	.updated-badge {
		padding: 0.25rem 0.5rem;
		background-color: #22c55e;
		color: white;
		border-radius: 4px;
		font-size: 0.75rem;
		font-weight: 600;
	}

	/* Custom styling for VTable columns */
	:global(.updated-indicator) {
		color: #dc3545;
		font-size: 1rem;
		margin-right: 0.25rem;
	}

	:global(.id-column) {
		font-weight: 600;
		color: #4042a3;
	}

	:global(.nombre-column) {
		font-weight: 500;
	}

	:global(.action-column) {
		width: 100px;
	}

	:global(.header-group) {
		background-color: #e9ecef !important;
		font-weight: 700;
	}

	:global(.text-center) {
		text-align: center;
	}

	:global(.btn-action) {
		padding: 0.25rem 0.5rem;
		background-color: #4042a3;
		color: white;
		border: none;
		border-radius: 4px;
		cursor: pointer;
		transition: background-color 0.2s;
		font-size: 0.875rem;
	}

	:global(.btn-action:hover) {
		background-color: #2f3280;
	}

	/* New snippet-based action buttons */
	:global(.action-buttons) {
		display: flex;
		gap: 0.25rem;
		justify-content: center;
		align-items: center;
	}

	:global(.btn-small) {
		padding: 0.25rem 0.5rem;
		border: none;
		background: #f3f4f6;
		border-radius: 4px;
		cursor: pointer;
		font-size: 0.875rem;
		transition: all 0.2s;
		display: inline-flex;
		align-items: center;
		justify-content: center;
	}

	:global(.btn-small:hover) {
		background: #e5e7eb;
		transform: scale(1.05);
	}

	:global(.btn-small:active) {
		transform: scale(0.95);
	}

	:global(.btn-edit:hover) {
		background: #dbeafe;
		color: #1e40af;
	}

	:global(.btn-delete:hover) {
		background: #fee2e2;
		color: #dc2626;
	}
</style>

