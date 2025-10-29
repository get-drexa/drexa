import type { Doc, Id } from "@fileone/convex/dataModel"
import { memo, useCallback } from "react"
import { TextFileIcon } from "@/components/icons/text-file-icon"
import { MiddleTruncatedText } from "@/components/ui/middle-truncated-text"
import { cn } from "@/lib/utils"

export type FileGridSelection = Set<Id<"files">>

export function FileGrid({
	files,
	selectedFiles = new Set(),
	onSelectionChange,
	onContextMenu,
}: {
	files: Doc<"files">[]
	selectedFiles?: FileGridSelection
	onSelectionChange?: (selection: FileGridSelection) => void
	onContextMenu?: (file: Doc<"files">, event: React.MouseEvent) => void
}) {
	const onItemSelect = useCallback(
		(file: Doc<"files">) => {
			onSelectionChange?.(new Set([file._id]))
		},
		[onSelectionChange],
	)

	const onItemContextMenu = useCallback(
		(file: Doc<"files">, event: React.MouseEvent) => {
			onContextMenu?.(file, event)
			onSelectionChange?.(new Set([file._id]))
		},
		[onContextMenu, onSelectionChange],
	)

	return (
		<div className="grid auto-cols-max grid-flow-col gap-3">
			{files.map((file) => (
				<FileGridItem
					selected={selectedFiles.has(file._id)}
					key={file._id}
					file={file}
					onSelect={onItemSelect}
					onContextMenu={onItemContextMenu}
				/>
			))}
		</div>
	)
}

const FileGridItem = memo(function FileGridItem({
	selected,
	file,
	onSelect,
	onContextMenu,
}: {
	selected: boolean
	file: Doc<"files">
	onSelect?: (file: Doc<"files">) => void
	onContextMenu?: (file: Doc<"files">, event: React.MouseEvent) => void
}) {
	return (
		<button
			type="button"
			key={file._id}
			className={cn(
				"flex flex-col gap-2 items-center justify-center w-24 p-[calc(var(--spacing)*1+1px)] rounded-md",
				{ "bg-muted border border-border p-1": selected },
			)}
			onClick={() => {
				onSelect?.(file)
			}}
			onContextMenu={(event) => {
				onContextMenu?.(file, event)
			}}
		>
			<TextFileIcon className="size-10" />
			<MiddleTruncatedText className="text-sm">
				{file.name}
			</MiddleTruncatedText>
		</button>
	)
})

export { FileGridItem }
