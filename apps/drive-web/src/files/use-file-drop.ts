import type { PrimitiveAtom } from "jotai"
import { useSetAtom, useStore } from "jotai"
import { useState } from "react"
import type { DirectoryInfo, DirectoryItem } from "@/vfs/vfs"

export interface FileDragInfo {
	source: DirectoryItem
	items: DirectoryItem[]
}

export interface UseFileDropOptions {
	destDir?: DirectoryInfo | string
	dragInfoAtom: PrimitiveAtom<FileDragInfo | null>
	enabled?: boolean
	onDrop?: (
		items: DirectoryItem[],
		targetDirectory: DirectoryInfo | string,
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
	destDir,
	dragInfoAtom,
	enabled,
	onDrop,
}: UseFileDropOptions): UseFileDropReturn {
	const [isDraggedOver, setIsDraggedOver] = useState(false)
	const setDragInfo = useSetAtom(dragInfoAtom)
	const store = useStore()

	const dirId = typeof destDir === "string" ? destDir : destDir?.id

	const handleDrop = (_e: React.DragEvent) => {
		if (!enabled || !destDir || !dirId || !onDrop) return

		const dragInfo = store.get(dragInfoAtom)
		if (dragInfo) {
			const items = dragInfo.items.filter((item) => item.id !== dirId)
			if (items.length > 0) {
				onDrop(items, destDir)
			}
		}
		setIsDraggedOver(false)
		setDragInfo(null)
	}

	const handleDragOver = (e: React.DragEvent) => {
		const dragInfo = store.get(dragInfoAtom)
		if (dragInfo && destDir) {
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
