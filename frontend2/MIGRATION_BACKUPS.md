# Backups Page Migration Summary

## Migration: Solid.js → Svelte 5

**Source:** `/home/ivanjoz/projects/genix/frontend/src/routes/admin/backups.tsx`  
**Target:** `/home/ivanjoz/projects/genix/frontend2/src/routes/admin/backups/`

---

## Files Created

### 1. `backups.svelte.ts` - Service Layer
- **Purpose:** Handles data fetching and state management using Svelte 5 runes
- **Key Features:**
  - `BackupsService` class extends `GetHandler` for cache-enabled fetching
  - Reactive `backups` array using `$state` rune
  - `createBackup()` function for generating new backups
  - `restoreBackup(name)` function for restoring from backup
  - `refreshBackups()` method to reload the list after operations

### 2. `+page.svelte` - UI Component
- **Purpose:** Main page component for backup management
- **Key Features:**
  - Two-column grid layout (backup list + restore panel)
  - VTable component for displaying backups with:
    - Created date column
    - Name column (highlighted)
    - Size column (formatted in MB)
    - Row selection support
  - Action buttons:
    - Create new backup (with confirmation)
    - Download selected backup to S3
    - Restore from selected backup (with confirmation)

---

## Key Changes & Conversions

### Solid.js → Svelte 5

| Aspect | Solid.js | Svelte 5 |
|--------|----------|----------|
| **Signals** | `createSignal()` | `$state()` rune |
| **API Hooks** | `useBackupsAPI()` | `BackupsService` class |
| **Components** | `QTable` | `VTable` |
| **Conditionals** | `<Show when={}>` | `{#if}` blocks |
| **Events** | `onclick={ev =>}` | `onclick={ev =>}` (same) |

### CSS Class Conversions (Solid.js custom → Tailwind)

| Original | Converted | Description |
|----------|-----------|-------------|
| `w100` | `w-full` | Full width |
| `jc-between` | `justify-between` | Flex justify |
| `ai-center` | `items-center` | Flex align items |
| `mb-06` | `mb-6` | Margin bottom |
| `mb-08` | `mb-8` | Margin bottom |
| `mr-08` | `mr-8` | Margin right |
| `bn1 s4 b-green` | `bx-green` | Green button |
| `bn1 s4 d-blue` | `bx-blue` | Blue button |
| `bn1 d-purple` | `bx-purple` | Purple button |
| `t-c` | `text-center` | Text center |

**Note:** Some classes like `ff-bold`, `ff-semibold`, `nowrap`, and `box-error-ms` are kept as-is since they exist in the frontend2 codebase.

---

## Features Implemented

### ✅ Backup List
- Displays all available backups in a table
- Shows creation date, name, and size
- Supports row selection (click to select/deselect)
- Uses VTable with virtualization for performance

### ✅ Create Backup
- Button with confirmation dialog
- Shows loading state during creation
- Refreshes list after successful creation
- Error handling with notifications

### ✅ Download Backup
- Downloads backup file from S3
- Uses signed URL from `Env.S3_URL`
- Opens in new tab for download

### ✅ Restore Backup
- Only enabled when a backup is selected
- Shows confirmation dialog with backup date
- Loading state during restore operation
- Error handling with notifications

---

## Architecture Patterns Used

### Service Pattern (GetHandler)
Following the frontend2 architecture documented in README.md:
- Service file uses `.svelte.ts` extension
- Located in same directory as `+page.svelte`
- Extends `GetHandler` class for cache support
- Uses `$state()` rune for reactive properties
- Cache configuration: `{ min: 0, ver: 1 }` (no cache for real-time data)

### Component Structure
- Uses `Page` component as container (from frontend2 components)
- VTable for data display with:
  - Column definitions using `ITableColumn<T>` interface
  - Row selection support
  - Click handlers
- Modal confirmations using `ConfirmWarn` helper

### HTTP Operations
- `POST` function for create/restore operations
- Automatic error handling via POST config
- Success/error messages configured in service layer
- Route refresh after mutations

---

## Testing Checklist

- [ ] Backup list loads correctly
- [ ] Create backup button shows confirmation
- [ ] Create backup refreshes list on success
- [ ] Row selection works (click to select/deselect)
- [ ] Download button downloads from correct S3 path
- [ ] Restore button shows confirmation with date
- [ ] Restore operation completes successfully
- [ ] Error messages display for failed operations
- [ ] Loading states show during async operations
- [ ] Responsive layout works on mobile

---

## Dependencies

### Imports from frontend2 core:
- `Page` - Page container component
- `VTable` - Virtualized table component
- `GetHandler` - Service base class
- `POST` - HTTP POST helper
- `Loading`, `Notify`, `ConfirmWarn` - UI helpers
- `formatTime` - Date formatting
- `formatN` - Number formatting

### External:
- `notiflix` - Loading indicators

---

## Notes

1. **Cache Strategy:** Backups use `min: 0` cache to ensure real-time data
2. **S3 Integration:** Download URLs constructed from `Env.S3_URL + "backups/1/{Name}"`
3. **Accessibility:** Added `aria-label` attributes to icon-only buttons
4. **Type Safety:** Full TypeScript typing with `IBackup` interface
5. **Reactivity:** Uses Svelte 5 runes (`$state`, `$derived`) for reactive state

