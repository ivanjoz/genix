/**
 * Virtualizer library exports
 *
 * This module provides a custom virtual scrolling implementation
 * inspired by TanStack Virtual, adapted for Svelte 5.
 */

// Core virtualizer
export { createVirtualizer } from './index.svelte';
export type { VirtualItem, VirtualizerOptions, VirtualizerStore } from './index.svelte';

// Types
export type { ITableColumn, CellRendererFn, CellRendererSnippet } from './types';
