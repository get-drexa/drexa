import { api } from "@fileone/convex/_generated/api"
import type { Doc, Id } from "@fileone/convex/_generated/dataModel"
import type {
	DirectoryHandle,
	FileSystemHandle,
} from "@fileone/convex/model/filesystem"
import { useMutation } from "@tanstack/react-query"
import { useMutation as useContextMutation } from "convex/react"
import type { Atom } from "jotai"
import { useStore } from "jotai"
import { useState } from "react"
import { toast } from "sonner"

export interface FileDragInfo {
	source: FileSystemHandle
	items: FileSystemHandle[]
}

export interface UseFileDropOptions {
	item: DirectoryHandle | null
	dragInfoAtom: Atom<FileDragInfo | null>
	onDropSuccess?: (
		items: Id<"files">[],
		targetDirectory: Doc<"directories">,
	) => void
}

export interface UseFileDropReturn {
	isDraggedOver: boolean
	dropHandlers: {
		onDrop: (e: React.DragEvent) => void
		onDragOver: (e: React.DragEvent) => void
		onDragLeave: () => void
	}
}

export function useFileDrop({
	item,
	dragInfoAtom,
}: UseFileDropOptions): UseFileDropReturn {
	const [isDraggedOver, setIsDraggedOver] = useState(false)
	const store = useStore()

	const { mutate: moveDroppedItems } = useMutation({
		mutationFn: useContextMutation(api.filesystem.moveItems),
		onSuccess: ({
			items,
			targetDirectory,
		}: {
			items: FileSystemHandle[]
			targetDirectory: Doc<"directories">
		}) => {
			toast.success(
				`${items.length} items moved to ${targetDirectory.name}`,
			)
		},
	})

	const handleDrop = (_e: React.DragEvent) => {
		const dragInfo = store.get(dragInfoAtom)
		if (dragInfo && item) {
			moveDroppedItems({
				targetDirectory: item,
				items: dragInfo.items,
			})
		}
		setIsDraggedOver(false)
	}

	const handleDragOver = (e: React.DragEvent) => {
		const dragInfo = store.get(dragInfoAtom)
		if (dragInfo && item) {
			e.preventDefault()
			e.dataTransfer.dropEffect = "move"
			setIsDraggedOver(true)
		} else {
			e.dataTransfer.dropEffect = "none"
		}
	}

	const handleDragLeave = () => {
		setIsDraggedOver(false)
	}

	return {
		isDraggedOver,
		dropHandlers: {
			onDrop: handleDrop,
			onDragOver: handleDragOver,
			onDragLeave: handleDragLeave,
		},
	}
}
