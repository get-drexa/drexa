import { api } from "@fileone/convex/_generated/api"
import type { Doc, Id } from "@fileone/convex/_generated/dataModel"
import type { DirectoryItem } from "@fileone/convex/model/directories"
import { useMutation } from "@tanstack/react-query"
import { useMutation as useContextMutation } from "convex/react"
import type { Atom } from "jotai"
import { useStore } from "jotai"
import { useState } from "react"
import { toast } from "sonner"

export interface FileDragInfo {
	source: DirectoryItem
	items: Id<"files">[]
}

export interface UseFileDropOptions {
	item: DirectoryItem
	dragInfoAtom: Atom<FileDragInfo | null>
	onDropSuccess?: (items: Id<"files">[], targetDirectory: Doc<"directories">) => void
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
	onDropSuccess,
}: UseFileDropOptions): UseFileDropReturn {
	const [isDraggedOver, setIsDraggedOver] = useState(false)
	const store = useStore()

	const { mutate: moveFiles } = useMutation({
		mutationFn: useContextMutation(api.files.moveFiles),
		onSuccess: ({
			items,
			targetDirectory,
		}: {
			items: Id<"files">[]
			targetDirectory: Doc<"directories">
		}) => {
			toast.success(
				`${items.length} files moved to ${targetDirectory.name}`,
			)
			onDropSuccess?.(items, targetDirectory)
		},
	})

	const handleDrop = (_e: React.DragEvent) => {
		const dragInfo = store.get(dragInfoAtom)
		if (dragInfo && item.kind === "directory") {
			moveFiles({
				targetDirectoryId: item.doc._id,
				items: dragInfo.items,
			})
		}
		setIsDraggedOver(false)
	}

	const handleDragOver = (e: React.DragEvent) => {
		const dragInfo = store.get(dragInfoAtom)
		if (
			dragInfo &&
			dragInfo.source !== item &&
			item.kind === "directory"
		) {
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