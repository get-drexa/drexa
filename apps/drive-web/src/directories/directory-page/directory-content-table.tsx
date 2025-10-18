import type { Doc } from "@fileone/convex/dataModel"
import {
	type DirectoryHandle,
	type FileHandle,
	type FileSystemHandle,
	type FileSystemItem,
	FileType,
	isSameHandle,
	newDirectoryHandle,
	newFileHandle,
	newFileSystemHandle,
} from "@fileone/convex/filesystem"
import { Link, useNavigate } from "@tanstack/react-router"
import {
	type ColumnDef,
	flexRender,
	getCoreRowModel,
	type Row,
	type Table as TableType,
	useReactTable,
} from "@tanstack/react-table"
import { type PrimitiveAtom, useSetAtom, useStore } from "jotai"
import { useContext, useEffect, useMemo, useRef } from "react"
import { DirectoryIcon } from "@/components/icons/directory-icon"
import { Checkbox } from "@/components/ui/checkbox"
import {
	Table,
	TableBody,
	TableCell,
	TableHead,
	TableHeader,
	TableRow,
} from "@/components/ui/table"
import {
	isControlOrCommandKeyActive,
	keyboardModifierAtom,
} from "@/lib/keyboard"
import { TextFileIcon } from "../../components/icons/text-file-icon"
import { type FileDragInfo, useFileDrop } from "../../files/use-file-drop"
import { cn } from "../../lib/utils"
import { DirectoryPageContext } from "./context"

type DirectoryContentTableProps = {
	filterFn: (item: FileSystemItem) => boolean
	directoryUrlFn: (directory: Doc<"directories">) => string
	fileDragInfoAtom: PrimitiveAtom<FileDragInfo | null>
	onContextMenu: (
		row: Row<FileSystemItem>,
		table: TableType<FileSystemItem>,
	) => void
	onOpenFile: (file: Doc<"files">) => void
}

function formatFileSize(bytes: number): string {
	if (bytes === 0) return "0 B"

	const k = 1024
	const sizes = ["B", "KB", "MB", "GB", "TB", "PB"]
	const i = Math.floor(Math.log(bytes) / Math.log(k))

	return `${parseFloat((bytes / k ** i).toFixed(2))} ${sizes[i]}`
}

function useTableColumns(
	onOpenFile: (file: Doc<"files">) => void,
	directoryUrlFn: (directory: Doc<"directories">) => string,
): ColumnDef<FileSystemItem>[] {
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
						case FileType.File:
							return (
								<FileNameCell
									file={row.original.doc}
									onOpenFile={onOpenFile}
								/>
							)
						case FileType.Directory:
							return (
								<DirectoryNameCell
									directory={row.original.doc}
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
						case FileType.File:
							return (
								<div>
									{formatFileSize(row.original.doc.size)}
								</div>
							)
						case FileType.Directory:
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
							{new Date(
								row.original.doc.createdAt,
							).toLocaleString()}
						</div>
					)
				},
			},
		],
		[onOpenFile, directoryUrlFn],
	)
}

export function DirectoryContentTable({
	filterFn,
	directoryUrlFn,
	onContextMenu,
	fileDragInfoAtom,
	onOpenFile,
}: DirectoryContentTableProps) {
	const { directoryContent } = useContext(DirectoryPageContext)
	const store = useStore()
	const navigate = useNavigate()

	const table = useReactTable({
		data: directoryContent || [],
		columns: useTableColumns(onOpenFile, directoryUrlFn),
		getCoreRowModel: getCoreRowModel(),
		enableRowSelection: true,
		enableGlobalFilter: true,
		globalFilterFn: (row, _columnId, _filterValue, _addMeta) =>
			filterFn(row.original),
		getRowId: (row) => row.doc._id,
	})

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

	const handleRowContextMenu = (
		row: Row<FileSystemItem>,
		_event: React.MouseEvent,
	) => {
		if (!row.getIsSelected()) {
			selectRow(row)
		}
		onContextMenu(row, table)
	}

	const selectRow = (row: Row<FileSystemItem>) => {
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

	const handleRowDoubleClick = (row: Row<FileSystemItem>) => {
		if (row.original.kind === FileType.Directory) {
			navigate({
				to: `/directories/${row.original.doc._id}`,
			})
		}
	}

	return (
		<div className="overflow-hidden">
			<Table>
				<TableHeader>
					{table.getHeaderGroups().map((headerGroup) => (
						<TableRow className="px-4" key={headerGroup.id}>
							{headerGroup.headers.map((header) => (
								<TableHead
									className="first:pl-4 last:pr-4"
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
				<TableBody>
					{table.getRowModel().rows?.length ? (
						table.getRowModel().rows.map((row) => (
							<FileItemRow
								key={row.id}
								table={table}
								row={row}
								onClick={() => selectRow(row)}
								fileDragInfoAtom={fileDragInfoAtom}
								onContextMenu={(e) =>
									handleRowContextMenu(row, e)
								}
								onDoubleClick={() => {
									handleRowDoubleClick(row)
								}}
							/>
						))
					) : (
						<NoResultsRow />
					)}
				</TableBody>
			</Table>
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
}: {
	table: TableType<FileSystemItem>
	row: Row<FileSystemItem>
	onClick: () => void
	onContextMenu: (e: React.MouseEvent) => void
	onDoubleClick: () => void
	fileDragInfoAtom: PrimitiveAtom<FileDragInfo | null>
}) {
	const ref = useRef<HTMLTableRowElement>(null)
	const setFileDragInfo = useSetAtom(fileDragInfoAtom)

	const { isDraggedOver, dropHandlers } = useFileDrop({
		destItem:
			row.original.kind === FileType.Directory
				? newDirectoryHandle(row.original.doc._id)
				: null,
		dragInfoAtom: fileDragInfoAtom,
	})

	const handleDragStart = (_e: React.DragEvent) => {
		let source: DirectoryHandle | FileHandle
		switch (row.original.kind) {
			case FileType.File:
				source = newFileHandle(row.original.doc._id)
				break
			case FileType.Directory:
				source = newDirectoryHandle(row.original.doc._id)
				break
		}

		let draggedItems: FileSystemHandle[]
		// drag all selections, but only if the currently dragged row is also selected
		if (row.getIsSelected()) {
			draggedItems = table
				.getSelectedRowModel()
				.rows.map((row) => newFileSystemHandle(row.original))
			if (!draggedItems.some((item) => isSameHandle(item, source))) {
				draggedItems.push(source)
			}
		} else {
			draggedItems = [source]
		}

		setFileDragInfo({
			source,
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
	directory: Doc<"directories">
	directoryUrlFn: (directory: Doc<"directories">) => string
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
	file: Doc<"files">
	onOpenFile: (file: Doc<"files">) => void
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
