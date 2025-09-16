import { api } from "@convex/_generated/api"
import type { Id } from "@convex/_generated/dataModel"
import type { DirectoryItem } from "@convex/model/directories"
import { useMutation } from "@tanstack/react-query"
import {
	type ColumnDef,
	flexRender,
	getCoreRowModel,
	type Row,
	useReactTable,
} from "@tanstack/react-table"
import { useMutation as useContextMutation, useQuery } from "convex/react"
import { useAtom, useAtomValue, useSetAtom, useStore } from "jotai"
import { CheckIcon, TextCursorInputIcon, TrashIcon, XIcon } from "lucide-react"
import { useEffect, useId, useRef } from "react"
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
import { TextFileIcon } from "../components/icons/text-file-icon"
import { Button } from "../components/ui/button"
import { LoadingSpinner } from "../components/ui/loading-spinner"
import { withDefaultOnError } from "../lib/error"
import { cn } from "../lib/utils"
import {
	contextMenuTargeItemAtom,
	newItemKindAtom,
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
				onCheckedChange={(value) =>
					table.toggleAllPageRowsSelected(!!value)
				}
				aria-label="Select all"
			/>
		),
		cell: ({ row }) => (
			<Checkbox
				checked={row.getIsSelected()}
				onCheckedChange={(value) => row.toggleSelected(!!value)}
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
					return <FileNameCell initialName={row.original.doc.name} />
				case "directory":
					return (
						<div className="flex w-full items-center gap-2">
							<DirectoryIcon className="size-4" />
							{row.original.doc.name}
						</div>
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

export function FileTable() {
	return (
		<FileTableContextMenu>
			<div className="w-full">
				<FileTableContent />
			</div>
		</FileTableContextMenu>
	)
}

export function FileTableContextMenu({
	children,
}: {
	children: React.ReactNode
}) {
	const store = useStore()
	const setOptimisticDeletedItems = useSetAtom(optimisticDeletedItemsAtom)
	const moveToTrashMutation = useContextMutation(api.files.moveToTrash)
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
			console.log("Renaming:", selectedItem.doc.name)
			// TODO: Implement rename functionality
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
		<ContextMenu>
			<ContextMenuTrigger asChild>{children}</ContextMenuTrigger>
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
		</ContextMenu>
	)
}

export function FileTableContent() {
	const directory = useQuery(api.files.fetchDirectoryContent, {})
	const optimisticDeletedItems = useAtomValue(optimisticDeletedItemsAtom)
	const setContextMenuTargetItem = useSetAtom(contextMenuTargeItemAtom)

	const handleRowContextMenu = (
		row: Row<DirectoryItem>,
		event: React.MouseEvent,
	) => {
		setContextMenuTargetItem(row.original)
	}

	const table = useReactTable({
		data: directory || [],
		columns,
		getCoreRowModel: getCoreRowModel(),
		enableRowSelection: true,
		enableGlobalFilter: true,
		globalFilterFn: (row, _columnId, _filterValue, _addMeta) => {
			return !optimisticDeletedItems.has(row.original.doc._id)
		},
	})

	if (!directory) {
		return null
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
							<TableRow
								key={row.id}
								data-state={row.getIsSelected() && "selected"}
								onContextMenu={(e) => {
									handleRowContextMenu(row, e)
								}}
							>
								{row.getVisibleCells().map((cell) => (
									<TableCell
										className="first:pl-4 last:pr-4"
										key={cell.id}
										style={{ width: cell.column.getSize() }}
									>
										{flexRender(
											cell.column.columnDef.cell,
											cell.getContext(),
										)}
									</TableCell>
								))}
							</TableRow>
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
			createDirectory({ name: itemName })
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

function FileNameCell({ initialName }: { initialName: string }) {
	return (
		<div className="flex w-full items-center gap-2">
			<TextFileIcon className="size-4" />
			{initialName}
		</div>
	)
}
