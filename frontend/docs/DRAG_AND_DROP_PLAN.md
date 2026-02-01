# Drag and Drop Implementation Plan: Ecommerce Builder

This document outlines the strategy for implementing drag-and-drop functionality in the Genix Store Builder.

## 1. Requirements
- **Source**: Drag templates from the `SectionEditorLayer` (specifically `TemplatesTab.svelte`).
- **Reordering**: Drag and reorder existing sections within the `EcommerceBuilder.svelte` canvas.
- **Visual Feedback**: 
  - A 48px height "ghost" placeholder showing the insertion point.
  - Highlighted state for the dragged element.
- **Technology**: Native HTML5 Drag and Drop API (No external dependencies).
- **Constraint**: The canvas will always contain at least one element.

## 2. Rationale
- **Performance**: Native API is lightweight and has zero bundle size impact.
- **Svelte 5 Compatibility**: Avoids potential conflicts between third-party DND libraries and Svelte 5's new reactivity system (Runes).
- **Flexibility**: Total control over the insertion logic and visual feedback without fighting library-specific abstractions.

## 3. Step-by-Step Implementation Plan

### Step 1: Data Transfer & Types
- Define a consistent data format for `event.dataTransfer`.
- Types of drag operations:
  - `application/x-genix-template`: For new sections from the sidebar.
  - `application/x-genix-reorder`: For moving existing sections.

### Step 2: Draggable Templates (`TemplatesTab.svelte`)
- Add `draggable="true"` to `.template-card`.
- Implement `ondragstart`:
  - Set `effectAllowed = 'copy'`.
  - Serialize the `template` object into the dataTransfer.
- Add visual styles for the dragging state using CSS `:active` or a dedicated class.

### Step 3: Draggable Sections (`EcommerceBuilder.svelte`)
- Update the `renderSection` snippet.
- Add `draggable="true"` to the `.section-wrapper`.
- Implement `ondragstart`:
  - Set `effectAllowed = 'move'`.
  - Store the `index` of the section being dragged.
  - Set a "dragging" state to dim the original element.

### Step 4: Drop Zone Logic (`EcommerceBuilder.svelte`)
- Create a reusable `DropZone` logic or component.
- The canvas will calculate the nearest insertion index based on the mouse position relative to existing sections.
- **Drop Zones**: 
  - Above the first section.
  - Between every section.
  - Below the last section.

### Step 5: Visual Ghost Placeholder
- Implement a reactive state `dropIndex` in `EcommerceBuilder`.
- In the `#each` loop of the canvas, render a placeholder `div` (48px height, dashed border) when `dragOver` is active at a specific index.
- Use `transition:height` for a smooth entry of the placeholder.

### Step 6: Handling the Drop
- Implement `ondragover`: Call `event.preventDefault()` to allow the drop.
- Implement `ondrop`:
  - Identify the source (New vs. Reorder).
  - If **New**: Parse template, convert to `ComponentAST`, and `splice` into the `elements` array.
  - If **Reorder**: Remove from `oldIndex` and `splice` into `newIndex`.
- Reset all drag-related states (`dropIndex`, `isDragging`, etc.).

### Step 7: Finalization
- Trigger `onUpdate` to notify parent components of the change.
- Ensure the `selectedSection` state is updated or cleared if the selected element was moved.

## 4. Technical Considerations
- **Event Delegation**: Use `stopPropagation` where necessary to prevent nested drag triggers.
- **Z-Index**: Ensure the ghost placeholder and drag indicators appear above the section content.
- **Accessibility**: Ensure sections remain keyboard-interactable (already implemented with `onclick` and `onkeydown`).
