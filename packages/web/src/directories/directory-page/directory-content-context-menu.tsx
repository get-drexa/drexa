import { api } from "@fileone/convex/_generated/api"
import { newFileSystemHandle } from "@fileone/convex/model/filesystem"
import { useMutation } from "@tanstack/react-query"
import { useMutation as useContextMutation } from "convex/react"
import { useAtom, useAtomValue, useSetAtom, useStore } from "jotai"
import { TextCursorInputIcon, TrashIcon } from "lucide-react"
import { toast } from "sonner"
import {
	ContextMenu,
	ContextMenuContent,
	ContextMenuItem,
	ContextMenuTrigger,
} from "@/components/ui/context-menu"
import {
	contextMenuTargeItemsAtom,
	itemBeingRenamedAtom,
	optimisticDeletedItemsAtom,
} from "./state"

export function DirectoryContentContextMenu({
	children,
}: {
	children: React.ReactNode
}) {
	const store = useStore()
	const [target, setTarget] = useAtom(contextMenuTargeItemsAtom)
	const setOptimisticDeletedItems = useSetAtom(optimisticDeletedItemsAtom)
	const moveToTrashMutation = useContextMutation(api.filesystem.moveToTrash)
	const { mutate: moveToTrash } = useMutation({
		mutationFn: moveToTrashMutation,
		onMutate: ({ handles }) => {
			setOptimisticDeletedItems(
				(prev) =>
					new Set([...prev, ...handles.map((handle) => handle.id)]),
			)
		},
		onSuccess: ({ deleted, errors }, { handles }) => {
			setOptimisticDeletedItems((prev) => {
				const newSet = new Set(prev)
				for (const handle of handles) {
					newSet.delete(handle.id)
				}
				return newSet
			})
			if (errors.length === 0 && deleted.length === handles.length) {
				toast.success(`Moved ${handles.length} items to trash`)
			} else if (errors.length === handles.length) {
				toast.error("Failed to move to trash")
			} else {
				toast.info(
					`Moved ${deleted.length} items to trash; failed to move ${errors.length} items`,
				)
			}
		},
	})

	const handleDelete = () => {
		const selectedItems = store.get(contextMenuTargeItemsAtom)
		if (selectedItems.length > 0) {
			moveToTrash({
				handles: selectedItems.map(newFileSystemHandle),
			})
		}
	}

	return (
		<ContextMenu
			onOpenChange={(open) => {
				if (!open) {
					setTarget([])
				}
			}}
		>
			<ContextMenuTrigger asChild>{children}</ContextMenuTrigger>
			{target && (
				<ContextMenuContent>
					<RenameMenuItem />
					<ContextMenuItem onClick={handleDelete}>
						<TrashIcon />
						Move to trash
					</ContextMenuItem>
				</ContextMenuContent>
			)}
		</ContextMenu>
	)
}

function RenameMenuItem() {
	const store = useStore()
	const target = useAtomValue(contextMenuTargeItemsAtom)
	const setItemBeingRenamed = useSetAtom(itemBeingRenamedAtom)

	const handleRename = () => {
		const selectedItems = store.get(contextMenuTargeItemsAtom)
		if (selectedItems.length === 1) {
			// biome-ignore lint/style/noNonNullAssertion: length is checked
			const selectedItem = selectedItems[0]!
			setItemBeingRenamed({
				originalItem: selectedItem,
				name: selectedItem.doc.name,
			})
		}
	}

	// Only render if exactly one item is selected
	if (target.length !== 1) {
		return null
	}

	return (
		<ContextMenuItem onClick={handleRename}>
			<TextCursorInputIcon />
			Rename
		</ContextMenuItem>
	)
}