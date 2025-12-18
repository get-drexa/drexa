/**
 * Mock component for testing virtualization with large datasets.
 *
 * Usage:
 * 1. Import this component instead of DirectoryContentTable
 * 2. Adjust TOTAL_ITEMS and ITEMS_PER_PAGE constants as needed
 * 3. The component will automatically use mock data
 */

import { infiniteQueryOptions, useInfiniteQuery } from "@tanstack/react-query"
import { useNavigate } from "@tanstack/react-router"
import {
	type ColumnDef,
	flexRender,
	getCoreRowModel,
	getFilteredRowModel,
	type Row,
	type Table as TableType,
	useReactTable,
} from "@tanstack/react-table"
import { useVirtualizer, type VirtualItem } from "@tanstack/react-virtual"
import { type PrimitiveAtom, useSetAtom, useStore } from "jotai"
import type React from "react"
import { useEffect, useMemo, useRef } from "react"
import { DirectoryIcon } from "@/components/icons/directory-icon"
import { TextFileIcon } from "@/components/icons/text-file-icon"
import { Checkbox } from "@/components/ui/checkbox"
import {
	Table,
	TableBody,
	TableCell,
	TableContainer,
	TableHead,
	TableHeader,
	TableRow,
} from "@/components/ui/table"
import { type FileDragInfo, useFileDrop } from "@/files/use-file-drop"
import {
	isControlOrCommandKeyActive,
	keyboardModifierAtom,
} from "@/lib/keyboard"
import { cn } from "@/lib/utils"
import type { DirectoryInfo, DirectoryItem, FileInfo } from "@/vfs/vfs"

// Configuration - adjust these to test different scenarios
const TOTAL_ITEMS = 10_000 // Total number of items to simulate
const ITEMS_PER_PAGE = 100 // Items per page (for pagination)
const NETWORK_DELAY_MS = 50 // Simulated network delay in milliseconds

/**
 * Generates mock directory items for testing virtualization.
 */
function generateMockDirectoryItems(
	totalItems: number,
	itemsPerPage: number = 100,
	mixFilesAndDirs: boolean = true,
): DirectoryItem[] {
	const items: DirectoryItem[] = []
	const now = new Date().toISOString()

	for (let i = 0; i < totalItems; i++) {
		const isFile = mixFilesAndDirs ? i % 2 === 0 : false
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
	const allItems = generateMockDirectoryItems(totalItems, itemsPerPage).sort(
		(a, b) => {
			// First sort by kind (directories before files)
			if (a.kind !== b.kind) {
				return a.kind.localeCompare(b.kind) // 'directory' < 'file'
			}
			// Then by name (default sort)
			return a.name.localeCompare(b.name)
		},
	)

	return infiniteQueryOptions({
		queryKey: ["mock", "directories", "content", totalItems],
		initialPageParam: {
			orderBy: "name" as const,
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
			const endIndex = Math.min(startIndex + itemsPerPage, allItems.length)
			const pageItems = allItems.slice(startIndex, endIndex)

			// Items are already sorted globally, but we may need to re-sort per page
			// based on the orderBy parameter (for now, we only support name sorting)
			// The global sort already handles directories first, so we just return the slice
			return {
				items: pageItems,
				nextCursor:
					endIndex < allItems.length ? endIndex.toString() : undefined,
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

/**
 * Helper to log virtualization metrics for debugging.
 */
function logVirtualizationStats(
	virtualItems: Array<{ index: number; start: number; size: number }>,
	totalRows: number,
) {
	if (virtualItems.length === 0) {
		console.log("No virtual items rendered")
		return
	}

	const firstIndex = virtualItems[0]?.index ?? 0
	const lastIndex = virtualItems[virtualItems.length - 1]?.index ?? 0
	const renderedCount = virtualItems.length

	console.log("Virtualization Stats:", {
		totalRows,
		renderedRows: renderedCount,
		firstRenderedIndex: firstIndex,
		lastRenderedIndex: lastIndex,
		coverage: `${((renderedCount / totalRows) * 100).toFixed(2)}%`,
		viewportRange: `${firstIndex} - ${lastIndex}`,
	})
}

type MockDirectoryContentTableProps = {
	directoryUrlFn: (directory: DirectoryInfo) => string
	fileDragInfoAtom: PrimitiveAtom<FileDragInfo | null>
	onContextMenu: (
		row: Row<DirectoryItem>,
		table: TableType<DirectoryItem>,
	) => void
	onOpenFile: (file: FileInfo) => void
}

function formatFileSize(bytes: number): string {
	if (bytes === 0) return "0 B"

	const k = 1024
	const sizes = ["B", "KB", "MB", "GB", "TB", "PB"]
	const i = Math.floor(Math.log(bytes) / Math.log(k))

	return `${parseFloat((bytes / k ** i).toFixed(2))} ${sizes[i]}`
}

function useTableColumns(
	onOpenFile: (file: FileInfo) => void,
	directoryUrlFn: (directory: DirectoryInfo) => string,
): ColumnDef<DirectoryItem>[] {
	return useMemo(
		() => [
			{
				id: "select",
				header: ({ table }) => (
					<Checkbox
						checked={table.getIsAllPageRowsSelected()}
						onCheckedChange={(value) => {
							table.toggleAllPageRowsSelected(!!value)
						}}
						aria-label="Select all"
					/>
				),
				cell: ({ row }) => (
					<Checkbox
						checked={row.getIsSelected()}
						onClick={(e) => {
							e.stopPropagation()
						}}
						onCheckedChange={row.getToggleSelectedHandler()}
						aria-label="Select row"
					/>
				),
				enableSorting: false,
				enableHiding: false,
				size: 24,
			},
			{
				header: "Name",
				accessorKey: "doc.name",
				cell: ({ row }) => {
					switch (row.original.kind) {
						case "file":
							return (
								<FileNameCell
									file={row.original}
									onOpenFile={onOpenFile}
								/>
							)
						case "directory":
							return (
								<DirectoryNameCell
									directory={row.original}
									directoryUrlFn={directoryUrlFn}
								/>
							)
					}
				},
				size: 1000,
			},
			{
				header: "Size",
				accessorKey: "size",
				cell: ({ row }) => {
					switch (row.original.kind) {
						case "file":
							return (
								<div>{formatFileSize(row.original.size)}</div>
							)
						case "directory":
							return <div className="font-mono">-</div>
					}
				},
			},
			{
				header: "Created At",
				accessorKey: "createdAt",
				cell: ({ row }) => {
					return (
						<div>
							{new Date(row.original.createdAt).toLocaleString()}
						</div>
					)
				},
			},
		],
		[onOpenFile, directoryUrlFn],
	)
}

export function MockDirectoryContentTable({
	directoryUrlFn,
	onContextMenu,
	fileDragInfoAtom,
	onOpenFile,
}: MockDirectoryContentTableProps) {
	// Use mock query instead of real API
	const mockQuery = useMemo(
		() =>
			createMockDirectoryContentQuery(
				TOTAL_ITEMS,
				ITEMS_PER_PAGE,
				NETWORK_DELAY_MS,
			),
		[],
	)

	const {
		data: directoryContent,
		isLoading: isLoadingDirectoryContent,
		isFetchingNextPage: isFetchingMoreDirectoryItems,
		fetchNextPage: fetchMoreDirectoryItems,
		hasNextPage: hasMoreDirectoryItems,
	} = useInfiniteQuery(mockQuery)

	const store = useStore()
	const navigate = useNavigate()

	const table = useReactTable({
		data: useMemo(
			() => directoryContent?.pages.flatMap((page) => page.items) || [],
			[directoryContent],
		),
		columns: useTableColumns(onOpenFile, directoryUrlFn),
		getCoreRowModel: getCoreRowModel(),
		getFilteredRowModel: getFilteredRowModel(),
		enableRowSelection: true,
		enableGlobalFilter: true,
		globalFilterFn: (row, _columnId, filterValue: Set<string>, _addMeta) =>
			!filterValue.has(row.original.id),
		getRowId: (row) => row.id,
	})
	const { rows } = table.getRowModel()

	const containerRef = useRef<HTMLDivElement>(null)
	const virtualizer = useVirtualizer({
		count: rows.length,
		getScrollElement: () => containerRef.current,
		estimateSize: () => 36,
		overscan: 20,
	})
	const virtualItems = virtualizer.getVirtualItems()

	// Log virtualization stats for debugging
	useEffect(() => {
		if (rows.length > 0 && virtualItems.length > 0) {
			logVirtualizationStats(virtualItems, rows.length)
		}
	}, [virtualItems, rows.length])

	useEffect(() => {
		const lastVirtualItem = virtualItems.at(-1)
		if (
			lastVirtualItem &&
			lastVirtualItem.index >= rows.length - 1 &&
			hasMoreDirectoryItems &&
			!isFetchingMoreDirectoryItems
		) {
			fetchMoreDirectoryItems()
		}
	}, [
		virtualItems,
		rows.length,
		hasMoreDirectoryItems,
		isFetchingMoreDirectoryItems,
		fetchMoreDirectoryItems,
	])

	useEffect(
		function escapeToClearSelections() {
			const handleEscape = (e: KeyboardEvent) => {
				if (e.key === "Escape") {
					table.setRowSelection({})
				}
			}
			window.addEventListener("keydown", handleEscape)
			return () => window.removeEventListener("keydown", handleEscape)
		},
		[table.setRowSelection],
	)

	if (isLoadingDirectoryContent) {
		return (
			<div className="flex items-center justify-center h-full">
				<div className="text-muted-foreground">
					Loading {TOTAL_ITEMS.toLocaleString()} mock items...
				</div>
			</div>
		)
	}

	const handleRowContextMenu = (
		row: Row<DirectoryItem>,
		_event: React.MouseEvent,
	) => {
		if (!row.getIsSelected()) {
			selectRow(row)
		}
		onContextMenu(row, table)
	}

	const selectRow = (row: Row<DirectoryItem>) => {
		const keyboardModifiers = store.get(keyboardModifierAtom)
		const isMultiSelectMode = isControlOrCommandKeyActive(keyboardModifiers)
		const isRowSelected = row.getIsSelected()
		if (isRowSelected && isMultiSelectMode) {
			row.toggleSelected(false)
		} else if (isRowSelected && !isMultiSelectMode) {
			table.setRowSelection({
				[row.id]: true,
			})
			row.toggleSelected(true)
		} else if (!isRowSelected) {
			if (isMultiSelectMode) {
				row.toggleSelected(true)
			} else {
				table.setRowSelection({
					[row.id]: true,
				})
			}
		}
	}

	const handleRowDoubleClick = (row: Row<DirectoryItem>) => {
		if (row.original.kind === "directory") {
			navigate({
				to: `/directories/${row.original.id}`,
			})
		}
	}

	const renderRow = (virtualRow: VirtualItem, i: number) => {
		const row = rows[virtualRow.index]
		if (!row) return null
		return (
			<FileItemRow
				style={{
					height: virtualRow.size,
					transform: `translateY(${
						virtualRow.start - i * virtualRow.size
					}px)`,
				}}
				key={row.id}
				table={table}
				row={row}
				onClick={() => selectRow(row)}
				fileDragInfoAtom={fileDragInfoAtom}
				onContextMenu={(e) => handleRowContextMenu(row, e)}
				onDoubleClick={() => {
					handleRowDoubleClick(row)
				}}
			/>
		)
	}

	return (
		<div className="h-full flex flex-col">
			{/* Debug info banner */}
			<div className="bg-muted px-4 py-2 text-sm border-b">
				<span className="font-semibold">Test Mode:</span> Showing{" "}
				{rows.length.toLocaleString()} of {TOTAL_ITEMS.toLocaleString()}{" "}
				items | Rendered: {virtualItems.length} rows | Check console for
				virtualization stats
			</div>
			<TableContainer className="flex-1" ref={containerRef}>
				<Table className="h-full min-h-0">
					<TableHeader>
						{table.getHeaderGroups().map((headerGroup) => (
							<TableRow
								className="px-4 border-b-0!"
								key={headerGroup.id}
							>
								{headerGroup.headers.map((header) => (
									<TableHead
										className="first:pl-4 last:pr-4 sticky top-0 bg-background z-1 inset-shadow-[0_-1px_0_0_var(--border)]"
										key={header.id}
										style={{ width: header.getSize() }}
									>
										{header.isPlaceholder
											? null
											: flexRender(
													header.column.columnDef
														.header,
													header.getContext(),
												)}
									</TableHead>
								))}
							</TableRow>
						))}
					</TableHeader>
					<TableBody className="overflow-auto">
						{rows.length > 0 ? (
							virtualItems.map(renderRow)
						) : (
							<NoResultsRow />
						)}
					</TableBody>
				</Table>
			</TableContainer>
		</div>
	)
}

function NoResultsRow() {
	return (
		<TableRow className="hover:bg-transparent">
			<TableCell colSpan={4} className="text-center">
				No results.
			</TableCell>
		</TableRow>
	)
}

function FileItemRow({
	table,
	row,
	onClick,
	onContextMenu,
	onDoubleClick,
	fileDragInfoAtom,
	...rowProps
}: React.ComponentProps<typeof TableRow> & {
	table: TableType<DirectoryItem>
	row: Row<DirectoryItem>
	onClick: () => void
	onContextMenu: (e: React.MouseEvent) => void
	onDoubleClick: () => void
	fileDragInfoAtom: PrimitiveAtom<FileDragInfo | null>
}) {
	const ref = useRef<HTMLTableRowElement>(null)
	const setFileDragInfo = useSetAtom(fileDragInfoAtom)

	const { isDraggedOver, dropHandlers } = useFileDrop({
		enabled: row.original.kind === "directory",
		destDir: row.original.kind === "directory" ? row.original : undefined,
		dragInfoAtom: fileDragInfoAtom,
	})

	const handleDragStart = (_e: React.DragEvent) => {
		let draggedItems: DirectoryItem[]
		if (row.getIsSelected()) {
			draggedItems = []
			let currentRowFound = false
			for (const { original: item } of table.getSelectedRowModel().rows) {
				draggedItems.push(item)
				if (item.id === row.original.id) {
					currentRowFound = true
				}
			}
			if (!currentRowFound) {
				draggedItems.push(row.original)
			}
		} else {
			draggedItems = [row.original]
		}

		setFileDragInfo({
			source: row.original,
			items: draggedItems,
		})
	}

	const handleDragEnd = () => {
		setFileDragInfo(null)
	}

	return (
		<TableRow
			draggable
			ref={ref}
			key={row.id}
			data-state={row.getIsSelected() && "selected"}
			onClick={onClick}
			onDoubleClick={onDoubleClick}
			onContextMenu={onContextMenu}
			onDragStart={handleDragStart}
			onDragEnd={handleDragEnd}
			{...dropHandlers}
			className={cn({ "bg-muted": isDraggedOver })}
			{...rowProps}
		>
			{row.getVisibleCells().map((cell) => (
				<TableCell
					className="first:pl-4 last:pr-4"
					key={cell.id}
					style={{ width: cell.column.getSize() }}
				>
					{flexRender(cell.column.columnDef.cell, cell.getContext())}
				</TableCell>
			))}
		</TableRow>
	)
}

function DirectoryNameCell({
	directory,
}: {
	directory: DirectoryInfo
	directoryUrlFn: (directory: DirectoryInfo) => string
}) {
	return (
		<div className="flex w-full items-center gap-2">
			<DirectoryIcon className="size-4" />
			<span className="hover:underline">{directory.name}</span>
		</div>
	)
}

function FileNameCell({
	file,
	onOpenFile,
}: {
	file: FileInfo
	onOpenFile: (file: FileInfo) => void
}) {
	return (
		<div className="flex w-full items-center gap-2">
			<TextFileIcon className="size-4" />
			<button
				type="button"
				className="hover:underline cursor-pointer"
				onClick={() => {
					onOpenFile(file)
				}}
			>
				{file.name}
			</button>
		</div>
	)
}

