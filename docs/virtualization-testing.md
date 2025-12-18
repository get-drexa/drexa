# Virtualization Testing Guide

This guide explains how to test the virtualization of the `DirectoryContentTable` component with large datasets.

## Quick Start

The app includes a dev-mode-only toggle to switch between the real and mock table. No code changes needed!

1. **Enable Dev Mode**: Make sure you're running in development mode (`bun dev` or `npm run dev`)

2. **Toggle Mock Table**: 
   - Look for the "Mock: OFF/ON" button in the directory page header
   - Click it to toggle between real and mock data
   - The toggle state persists in localStorage

3. **Adjust Test Parameters** (optional): Edit `apps/drive-web/src/directories/directory-page/mock-directory-content-table.tsx`:
   ```tsx
   const TOTAL_ITEMS = 10_000      // Total number of items to simulate
   const ITEMS_PER_PAGE = 100      // Items per page (for pagination)
   const NETWORK_DELAY_MS = 50     // Simulated network delay in milliseconds
   ```

## How It Works

- The `DirectoryContentTableWrapper` component automatically handles switching between real and mock tables
- Uses Jotai's `atomWithStorage` to persist the toggle state in localStorage
- The mock table code is completely tree-shaken in production builds
- No page reload needed - switching is instant

## Test Configuration

You can adjust these parameters in `mock-directory-content-table.tsx`:

- **TOTAL_ITEMS**: Total number of items to simulate (default: 10,000)
  - Try: 1,000 for quick tests, 50,000+ for stress testing
  
- **ITEMS_PER_PAGE**: Number of items per page (default: 100)
  - This simulates pagination behavior
  
- **NETWORK_DELAY_MS**: Simulated network delay (default: 50ms)
  - Set to 0 for instant loading, higher values to test loading states

## What to Test

1. **Scroll Performance**: Scroll through the list and verify smooth scrolling
2. **Infinite Loading**: Scroll to the bottom and verify more items load automatically
3. **Selection**: Select multiple items and verify performance
4. **Virtualization Stats**: Check the browser console for virtualization metrics
5. **Memory Usage**: Monitor browser DevTools to ensure memory stays reasonable

## Debugging

The mock table component includes:
- A debug banner showing current stats (total items, rendered rows)
- Console logging of virtualization metrics (check browser console)
- Visual indicators for rendered vs total items

## Implementation Details

### Files

- `apps/drive-web/src/directories/directory-page/directory-content-table-wrapper.tsx`
  - Wrapper component that conditionally loads real or mock table
  - Uses Jotai atoms for state management
  - Exports `mockTableAtom` for toggle components

- `apps/drive-web/src/directories/directory-page/mock-directory-content-table.tsx`
  - Mock table component with all utilities inlined
  - Generates mock data matching API behavior (directories first, then files)
  - Includes virtualization stats logging

### Utilities

The `mock-directory-content-table.tsx` file includes all utilities inlined:

- `generateMockDirectoryItems()`: Generate mock data
- `createMockDirectoryContentQuery()`: Create a mock infinite query
- `logVirtualizationStats()`: Log virtualization metrics

All utilities are defined within the same file for convenience.

## Example Test Scenarios

### Small Dataset (1,000 items)
```tsx
const TOTAL_ITEMS = 1_000
const ITEMS_PER_PAGE = 100
```

### Medium Dataset (10,000 items)
```tsx
const TOTAL_ITEMS = 10_000
const ITEMS_PER_PAGE = 100
```

### Large Dataset (100,000 items)
```tsx
const TOTAL_ITEMS = 100_000
const ITEMS_PER_PAGE = 200
```

### Stress Test (1,000,000 items)
```tsx
const TOTAL_ITEMS = 1_000_000
const ITEMS_PER_PAGE = 500
```

## Production Safety

- The mock table code is completely excluded from production builds
- Vite's static analysis removes all dev-mode code paths
- The toggle button only appears in development mode
- No performance impact on production bundles

