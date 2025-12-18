/**
 * Mock component for testing virtualization with large datasets.
 *
 * Usage:
 * 1. Import this component instead of DirectoryContentTable
 * 2. Adjust TOTAL_ITEMS and ITEMS_PER_PAGE constants as needed
 * 3. The component will automatically use mock data
 */

import { type InfiniteData, infiniteQueryOptions } from "@tanstack/react-query"
import { atom } from "jotai"
import { useMemo } from "react"
import type { FileDragInfo } from "@/files/use-file-drop"
import type { DirectoryContentOrderBy } from "@/vfs/api"
import type { DirectoryInfo, DirectoryItem, FileInfo } from "@/vfs/vfs"
import {
	DirectoryContentTable,
	type DirectoryContentTableProps,
} from "./directory-content-table"

// Configuration - adjust these to test different scenarios
const TOTAL_ITEMS = 10_000 // Total number of items to simulate
const ITEMS_PER_PAGE = 100 // Items per page (for pagination)
const NETWORK_DELAY_MS = 50 // Simulated network delay in milliseconds

/**
 * Generates mock directory items for testing virtualization.
 */
function generateMockDirectoryItems(
	totalItems: number,
	_mixFilesAndDirs: boolean = true,
): DirectoryItem[] {
	const items: DirectoryItem[] = []
	const now = new Date()

	for (let i = 0; i < totalItems; i++) {
		const isFile = _mixFilesAndDirs ? i % 2 === 0 : false
		const id = `mock-${i}`
		const name = isFile
			? `file-${i.toString().padStart(4, "0")}.txt`
			: `directory-${i.toString().padStart(4, "0")}`

		if (isFile) {
			const file: FileInfo = {
				kind: "file",
				id,
				parentId: "mock-parent",
				name,
				size: Math.floor(Math.random() * 10_000_000), // Random size up to 10MB
				mimeType: "text/plain",
				createdAt: now,
				updatedAt: now,
			}
			items.push(file)
		} else {
			const directory: DirectoryInfo = {
				kind: "directory",
				id,
				parentId: "mock-parent",
				name,
				createdAt: now,
				updatedAt: now,
			}
			items.push(directory)
		}
	}

	return items
}

/**
 * Creates a mock infinite query options that simulates paginated directory content.
 * Matches the real API behavior: directories are always sorted first, then files.
 */
function createMockDirectoryContentQuery(
	totalItems: number = 10_000,
	itemsPerPage: number = 100,
	delayMs: number = 0,
) {
	// Generate all items and sort them to match API behavior:
	// 1. Directories first (kind='directory' < kind='file' alphabetically)
	// 2. Then by the sort field (name, createdAt, etc.)
	// 3. Then by id as tiebreaker
	const allItems = generateMockDirectoryItems(totalItems).sort((a, b) => {
		// First sort by kind (directories before files)
		if (a.kind !== b.kind) {
			return a.kind.localeCompare(b.kind) // 'directory' < 'file'
		}
		// Then by name (default sort)
		return a.name.localeCompare(b.name)
	})

	type MockPageData = { items: DirectoryItem[]; nextCursor?: string }
	type MockPageParam = {
		orderBy: DirectoryContentOrderBy
		direction: "asc" | "desc"
		limit: number
		cursor: string
	}

	return infiniteQueryOptions<
		MockPageData,
		Error,
		InfiniteData<MockPageData, MockPageParam>,
		readonly (string | undefined)[],
		MockPageParam
	>({
		queryKey: ["mock", "directories", "content", totalItems] as readonly (
			| string
			| undefined
		)[],
		initialPageParam: {
			orderBy: "name" as DirectoryContentOrderBy,
			direction: "asc" as const,
			limit: itemsPerPage,
			cursor: "",
		},
		queryFn: async ({ pageParam }) => {
			// Simulate network delay if specified
			if (delayMs > 0) {
				await new Promise((resolve) => setTimeout(resolve, delayMs))
			}

			const cursor = pageParam.cursor || ""
			const startIndex = cursor ? parseInt(cursor, 10) : 0
			const endIndex = Math.min(
				startIndex + itemsPerPage,
				allItems.length,
			)
			const pageItems = allItems.slice(startIndex, endIndex)

			// Items are already sorted globally, but we may need to re-sort per page
			// based on the orderBy parameter (for now, we only support name sorting)
			// The global sort already handles directories first, so we just return the slice
			const nextCursor =
				endIndex < allItems.length ? endIndex.toString() : undefined
			return {
				items: pageItems,
				...(nextCursor !== undefined && { nextCursor }),
			}
		},
		getNextPageParam: (lastPage, _pages, lastPageParam) =>
			lastPage.nextCursor
				? {
						...lastPageParam,
						cursor: lastPage.nextCursor,
					}
				: null,
	})
}

export type MockDirectoryContentTableProps = Omit<
	DirectoryContentTableProps,
	"query" | "loadingComponent" | "debugBanner" | "fileDragInfoAtom"
>

export function MockDirectoryContentTable({
	directoryUrlFn,
	onContextMenu,
	onOpenFile,
}: MockDirectoryContentTableProps) {
	// Use mock query options instead of real API
	const query = useMemo(
		() =>
			createMockDirectoryContentQuery(
				TOTAL_ITEMS,
				ITEMS_PER_PAGE,
				NETWORK_DELAY_MS,
			),
		[],
	)

	const fileDragInfoAtom = useMemo(() => atom<FileDragInfo | null>(null), [])

	// Create debug banner (will be updated after query runs)
	const debugBanner = (
		<div className="bg-muted px-4 py-2 text-sm border-b">
			<span className="font-semibold">Test Mode:</span> Simulating{" "}
			{TOTAL_ITEMS.toLocaleString()} items | Check console for
			virtualization stats
		</div>
	)

	const loadingComponent = (
		<div className="flex items-center justify-center h-full">
			<div className="text-muted-foreground">
				Loading {TOTAL_ITEMS.toLocaleString()} mock items...
			</div>
		</div>
	)

	return (
		<DirectoryContentTable
			directoryUrlFn={directoryUrlFn}
			onContextMenu={onContextMenu}
			fileDragInfoAtom={fileDragInfoAtom}
			onOpenFile={onOpenFile}
			query={query}
			loadingComponent={loadingComponent}
			debugBanner={debugBanner}
		/>
	)
}
