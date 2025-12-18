import { useInfiniteQuery } from "@tanstack/react-query"
import { Link, useNavigate, useSearch } from "@tanstack/react-router"
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
import { type PrimitiveAtom, useAtomValue, useSetAtom, useStore } from "jotai"
import type React from "react"
import { useContext, useEffect, useMemo, useRef } from "react"
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
import { directoryContentQueryAtom } from "../../vfs/api"
import { DirectoryPageContext } from "./context"
import { DirectoryContentTableSkeleton } from "./directory-content-table-skeleton"

type DirectoryContentTableItemIdFilter = Set<string>

type DirectoryContentTableProps = {
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

export function DirectoryContentTable({
	directoryUrlFn,
	onContextMenu,
	fileDragInfoAtom,
	onOpenFile,
}: DirectoryContentTableProps) {
	const { directory } = useContext(DirectoryPageContext)
	const search = useSearch({
		from: "/_authenticated/_sidebar-layout/directories/$directoryId",
	})

	const directoryContentQuery = useAtomValue(
		directoryContentQueryAtom({
			directoryId: directory.id,
			orderBy: search.orderBy,
			direction: search.direction,
			limit: 100,
		}),
	)
	const {
		data: directoryContent,
		isLoading: isLoadingDirectoryContent,
		isFetchingNextPage: isFetchingMoreDirectoryItems,
		fetchNextPage: fetchMoreDirectoryItems,
		hasNextPage: hasMoreDirectoryItems,
	} = useInfiniteQuery(directoryContentQuery)

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
		globalFilterFn: (
			row,
			_columnId,
			filterValue: DirectoryContentTableItemIdFilter,
			_addMeta,
		) => !filterValue.has(row.original.id),
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
		return <DirectoryContentTableSkeleton />
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
		const row = rows[virtualRow.index]!
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
		<TableContainer className="h-full" ref={containerRef}>
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
												header.column.columnDef.header,
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
		// drag all selections, but only if the currently dragged row is also selected
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
	directoryUrlFn,
}: {
	directory: DirectoryInfo
	directoryUrlFn: (directory: DirectoryInfo) => string
}) {
	return (
		<div className="flex w-full items-center gap-2">
			<DirectoryIcon className="size-4" />
			<Link className="hover:underline" to={directoryUrlFn(directory)}>
				{directory.name}
			</Link>
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
