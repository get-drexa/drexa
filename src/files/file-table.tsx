import { api } from "@convex/_generated/api"
import type { DirectoryItem } from "@convex/model/directories"
import type { ColumnDef, Row } from "@tanstack/react-table"
import { useQuery } from "convex/react"
import { TextCursorInputIcon, TrashIcon } from "lucide-react"
import { useState } from "react"
import { DirectoryIcon } from "@/components/icons/directory-icon"
import { Checkbox } from "@/components/ui/checkbox"
import {
	ContextMenu,
	ContextMenuContent,
	ContextMenuItem,
	ContextMenuTrigger,
} from "@/components/ui/context-menu"
import { DataTable } from "@/components/ui/data-table"

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
	const directory = useQuery(api.files.fetchDirectoryContent, {})
	const [selectedItem, setSelectedItem] = useState<DirectoryItem | null>(null)

	if (!directory) {
		return null
	}

	const handleRename = () => {
		if (selectedItem) {
			console.log("Renaming:", selectedItem.doc.name)
			// TODO: Implement rename functionality
		}
	}

	const handleDelete = () => {
		if (selectedItem) {
			console.log("Deleting:", selectedItem.doc.name)
			// TODO: Implement delete functionality
		}
	}

	const handleRowContextMenu = (row: Row<DirectoryItem>) => {
		setSelectedItem(row.original)
	}

	return (
		<ContextMenu>
			<ContextMenuTrigger asChild>
				<div className="w-full">
					<DataTable
						columns={columns}
						data={directory}
						onRowContextMenu={handleRowContextMenu}
					/>
				</div>
			</ContextMenuTrigger>
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

function FileNameCell({ initialName }: { initialName: string }) {
	return <div>{initialName}</div>
}
