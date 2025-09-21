import { api } from "@fileone/convex/_generated/api"
import type { Doc } from "@fileone/convex/_generated/dataModel"
import type { DirectoryItem } from "@fileone/convex/model/directories"
import {
	newDirectoryHandle,
	newFileHandle,
} from "@fileone/convex/model/filesystem"
import { useMutation } from "@tanstack/react-query"
import { Link, useNavigate } from "@tanstack/react-router"
import {
	type ColumnDef,
	flexRender,
	getCoreRowModel,
	type Row,
	useReactTable,
} from "@tanstack/react-table"
import { useMutation as useContextMutation } from "convex/react"
import { useAtom, useAtomValue, useSetAtom, useStore } from "jotai"
import { CheckIcon, TextCursorInputIcon, TrashIcon, XIcon } from "lucide-react"
import { useContext, useEffect, useId, useRef } from "react"
import { toast } from "sonner"
import { DirectoryIcon } from "@/components/icons/directory-icon"
import { Checkbox } from "@/components/ui/checkbox"
import {
	ContextMenu,
	ContextMenuContent,
	ContextMenuItem,
	ContextMenuTrigger,
} from "@/components/ui/context-menu"
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
import { Button } from "../../components/ui/button"
import { LoadingSpinner } from "../../components/ui/loading-spinner"
import { useFileDrop } from "../../files/use-file-drop"
import { withDefaultOnError } from "../../lib/error"
import { cn } from "../../lib/utils"
import { DirectoryPageContext } from "./context"
import {
	contextMenuTargeItemAtom,
	dragInfoAtom,
	itemBeingRenamedAtom,
	newItemKindAtom,
	openedFileAtom,
	optimisticDeletedItemsAtom,
} from "./state"

function formatFileSize(bytes: number): string {
	if (bytes === 0) return "0 B"

	const k = 1024
	const sizes = ["B", "KB", "MB", "GB", "TB", "PB"]
	const i = Math.floor(Math.log(bytes) / Math.log(k))

	return `${parseFloat((bytes / k ** i).toFixed(2))} ${sizes[i]}`
}

const columns: ColumnDef<DirectoryItem>[] = [
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
					return <FileNameCell file={row.original.doc} />
				case "directory":
					return <DirectoryNameCell directory={row.original.doc} />
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
					return <div>{formatFileSize(row.original.doc.size)}</div>
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
					{new Date(row.original.doc.createdAt).toLocaleString()}
				</div>
			)
		},
	},
]

export function DirectoryContentTable() {
	return (
		<DirectoryContentTableContextMenu>
			<div className="w-full">
				<DirectoryContentTableContent />
			</div>
		</DirectoryContentTableContextMenu>
	)
}

export function DirectoryContentTableContextMenu({
	children,
}: {
	children: React.ReactNode
}) {
	const store = useStore()
	const target = useAtomValue(contextMenuTargeItemAtom)
	const setOptimisticDeletedItems = useSetAtom(optimisticDeletedItemsAtom)
	const moveToTrashMutation = useContextMutation(api.files.moveToTrash)
	const setItemBeingRenamed = useSetAtom(itemBeingRenamedAtom)
	const setContextMenuTargetItem = useSetAtom(contextMenuTargeItemAtom)
	const { mutate: moveToTrash } = useMutation({
		mutationFn: moveToTrashMutation,
		onMutate: ({ itemId }) => {
			setOptimisticDeletedItems((prev) => new Set([...prev, itemId]))
		},
		onSuccess: (itemId) => {
			setOptimisticDeletedItems((prev) => {
				const newSet = new Set(prev)
				newSet.delete(itemId)
				return newSet
			})
			toast.success("Moved to trash")
		},
	})

	const handleRename = () => {
		const selectedItem = store.get(contextMenuTargeItemAtom)
		if (selectedItem) {
			setItemBeingRenamed({
				kind: selectedItem.kind,
				originalItem: selectedItem,
				name: selectedItem.doc.name,
			})
		}
	}

	const handleDelete = () => {
		const selectedItem = store.get(contextMenuTargeItemAtom)
		if (selectedItem) {
			moveToTrash({
				kind: selectedItem.kind,
				itemId: selectedItem.doc._id,
			})
		}
	}

	return (
		<ContextMenu
			onOpenChange={(open) => {
				if (!open) {
					setContextMenuTargetItem(null)
				}
			}}
		>
			<ContextMenuTrigger asChild>{children}</ContextMenuTrigger>
			{target && (
				<ContextMenuContent>
					<ContextMenuItem onClick={handleRename}>
						<TextCursorInputIcon />
						Rename
					</ContextMenuItem>
					<ContextMenuItem onClick={handleDelete}>
						<TrashIcon />
						Move to trash
					</ContextMenuItem>
				</ContextMenuContent>
			)}
		</ContextMenu>
	)
}

export function DirectoryContentTableContent() {
	const { directoryContent } = useContext(DirectoryPageContext)
	const optimisticDeletedItems = useAtomValue(optimisticDeletedItemsAtom)
	const setContextMenuTargetItem = useSetAtom(contextMenuTargeItemAtom)
	const store = useStore()
	const navigate = useNavigate()

	const handleRowContextMenu = (
		row: Row<DirectoryItem>,
		_event: React.MouseEvent,
	) => {
		const target = store.get(contextMenuTargeItemAtom)
		if (target === row.original) {
			setContextMenuTargetItem(null)
		} else {
			selectRow(row)
			setContextMenuTargetItem(row.original)
		}
	}

	const table = useReactTable({
		data: directoryContent || [],
		columns,
		getCoreRowModel: getCoreRowModel(),
		enableRowSelection: true,
		enableGlobalFilter: true,
		globalFilterFn: (row, _columnId, _filterValue, _addMeta) => {
			return !optimisticDeletedItems.has(row.original.doc._id)
		},
		getRowId: (row) => row.doc._id,
	})

	const selectRow = (row: Row<DirectoryItem>) => {
		const keyboardModifiers = store.get(keyboardModifierAtom)
		const isMultiSelectMode = isControlOrCommandKeyActive(keyboardModifiers)
		const isRowSelected = row.getIsSelected()
		console.log({ isMultiSelectMode, isRowSelected })
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
								row={row}
								onClick={() => selectRow(row)}
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
					<NewItemRow />
				</TableBody>
			</Table>
		</div>
	)
}

function NoResultsRow() {
	const newItemKind = useAtomValue(newItemKindAtom)
	if (newItemKind) {
		return null
	}
	return (
		<TableRow>
			<TableCell colSpan={columns.length} className="text-center">
				No results.
			</TableCell>
		</TableRow>
	)
}

function NewItemRow() {
	const { directory } = useContext(DirectoryPageContext)
	const inputRef = useRef<HTMLInputElement>(null)
	const newItemFormId = useId()
	const [newItemKind, setNewItemKind] = useAtom(newItemKindAtom)
	const { mutate: createDirectory, isPending } = useMutation({
		mutationFn: useContextMutation(api.files.createDirectory),
		onSuccess: () => {
			setNewItemKind(null)
		},
		onError: withDefaultOnError(() => {
			setTimeout(() => {
				inputRef.current?.focus()
			}, 1)
		}),
	})

	useEffect(() => {
		if (newItemKind) {
			setTimeout(() => {
				inputRef.current?.focus()
			}, 1)
		}
	}, [newItemKind])

	if (!newItemKind) {
		return null
	}

	const onSubmit = (event: React.FormEvent<HTMLFormElement>) => {
		event.preventDefault()

		const formData = new FormData(event.currentTarget)
		const itemName = formData.get("itemName") as string

		if (itemName) {
			createDirectory({ name: itemName, directoryId: directory._id })
		} else {
			toast.error("Please enter a name.")
		}
	}

	const clearNewItemKind = () => {
		// setItemBeingAdded(null)
		setNewItemKind(null)
	}

	return (
		<TableRow className={cn("align-middle", { "opacity-50": isPending })}>
			<TableCell />
			<TableCell className="p-0">
				<div className="flex items-center gap-2 px-2 py-1 h-full">
					{isPending ? (
						<LoadingSpinner className="size-6" />
					) : (
						<DirectoryIcon />
					)}
					<form
						className="w-full"
						id={newItemFormId}
						onSubmit={onSubmit}
					>
						<input
							ref={inputRef}
							type="text"
							name="itemName"
							defaultValue={newItemKind}
							disabled={isPending}
							className="w-full h-8 px-2 bg-transparent border border-input rounded-sm outline-none focus:border-primary focus:ring-1 focus:ring-primary"
						/>
					</form>
				</div>
			</TableCell>
			<TableCell />
			<TableCell align="right" className="space-x-2 p-1">
				{!isPending ? (
					<>
						<Button
							type="button"
							form={newItemFormId}
							variant="ghost"
							size="icon"
							onClick={clearNewItemKind}
						>
							<XIcon />
						</Button>
						<Button type="submit" form={newItemFormId} size="icon">
							<CheckIcon />
						</Button>
					</>
				) : null}
			</TableCell>
		</TableRow>
	)
}

function FileItemRow({
	row,
	onClick,
	onContextMenu,
	onDoubleClick,
}: {
	row: Row<DirectoryItem>
	onClick: () => void
	onContextMenu: (e: React.MouseEvent) => void
	onDoubleClick: () => void
}) {
	const ref = useRef<HTMLTableRowElement>(null)
	const setDragInfo = useSetAtom(dragInfoAtom)

	const { isDraggedOver, dropHandlers } = useFileDrop({
		item:
			row.original.kind === "directory"
				? newDirectoryHandle(row.original.doc._id)
				: null,
		dragInfoAtom,
	})

	const handleDragStart = (e: React.DragEvent) => {
		if (row.original.kind === "file") {
			e.dataTransfer.setData(
				"application/x-internal",
				JSON.stringify(row.original),
			)
			const fileHandle = newFileHandle(row.original.doc._id)
			setDragInfo({
				source: fileHandle,
				items: [fileHandle],
			})
		}
	}

	const handleDragEnd = () => {
		setDragInfo(null)
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

function DirectoryNameCell({ directory }: { directory: Doc<"directories"> }) {
	return (
		<div className="flex w-full items-center gap-2">
			<DirectoryIcon className="size-4" />
			<Link
				className="hover:underline"
				to={`/directories/${directory._id}`}
			>
				{directory.name}
			</Link>
		</div>
	)
}

function FileNameCell({ file }: { file: Doc<"files"> }) {
	const setOpenedFile = useSetAtom(openedFileAtom)

	return (
		<div className="flex w-full items-center gap-2">
			<TextFileIcon className="size-4" />
			<button
				type="button"
				className="hover:underline cursor-pointer"
				onClick={() => {
					setOpenedFile(file)
				}}
			>
				{file.name}
			</button>
		</div>
	)
}
