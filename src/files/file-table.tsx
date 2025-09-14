import { api } from "@convex/_generated/api"
import type { Id } from "@convex/_generated/dataModel"
import type {
	DirectoryItem,
	DirectoryItemKind,
} from "@convex/model/directories"
import { useMutation } from "@tanstack/react-query"
import {
	type ColumnDef,
	flexRender,
	getCoreRowModel,
	type Row,
	useReactTable,
} from "@tanstack/react-table"
import { useMutation as useContextMutation, useQuery } from "convex/react"
import { atom, useAtomValue, useSetAtom, useStore } from "jotai"
import { TextCursorInputIcon, TrashIcon } from "lucide-react"
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

function formatFileSize(bytes: number): string {
	if (bytes === 0) return "0 B"

	const k = 1024
	const sizes = ["B", "KB", "MB", "GB", "TB", "PB"]
	const i = Math.floor(Math.log(bytes) / Math.log(k))

	return `${parseFloat((bytes / k ** i).toFixed(2))} ${sizes[i]}`
}

const contextMenuTargeItemAtom = atom<DirectoryItem | null>(null)
const optimisticDeletedItemsAtom = atom(
	new Set<Id<"files"> | Id<"directories">>(),
)

const itemBeingAddedAtom = atom<{
	kind: DirectoryItemKind
	name: string
} | null>(null)

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
						<div className="font-mono">
							<DirectoryIcon />
							{row.original.doc.name}
						</div>
					)
			}
		},
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
	const moveToTrash = useContextMutation(api.files.moveToTrash)
	const moveToTrashMutation = useMutation({
		mutationFn: (itemId: Id<"files"> | Id<"directories">) =>
			moveToTrash({ itemId }),
		onMutate: (itemId) => {
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
			moveToTrashMutation.mutate(selectedItem.doc._id)
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
		<div className="overflow-hidden rounded-md border">
			<Table>
				<TableHeader>
					{table.getHeaderGroups().map((headerGroup) => (
						<TableRow key={headerGroup.id}>
							{headerGroup.headers.map((header) => (
								<TableHead key={header.id}>
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
									<TableCell key={cell.id}>
										{flexRender(
											cell.column.columnDef.cell,
											cell.getContext(),
										)}
									</TableCell>
								))}
							</TableRow>
						))
					) : (
						<TableRow>
							<TableCell
								colSpan={columns.length}
								className="text-center"
							>
								No results.
							</TableCell>
						</TableRow>
					)}
					<NewItemRow />
				</TableBody>
			</Table>
		</div>
	)
}

function NewItemRow() {
	const itemBeingAdded = useAtomValue(itemBeingAddedAtom)
	if (!itemBeingAdded) {
		return null
	}
	return (
		<TableRow>
			<TableCell />
			<TableCell>
				<input type="text" value={itemBeingAdded.name} />
			</TableCell>
		</TableRow>
	)
}

function FileNameCell({ initialName }: { initialName: string }) {
	return <div>{initialName}</div>
}
