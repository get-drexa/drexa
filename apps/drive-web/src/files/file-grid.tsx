import { memo, useCallback } from "react"
import { TextFileIcon } from "@/components/icons/text-file-icon"
import { MiddleTruncatedText } from "@/components/ui/middle-truncated-text"
import { cn } from "@/lib/utils"
import type { FileInfo } from "./file"

export type FileGridSelection = Set<string>

export function FileGrid({
	files,
	selectedFiles = new Set(),
	onSelectionChange,
	onContextMenu,
}: {
	files: FileInfo[]
	selectedFiles?: FileGridSelection
	onSelectionChange?: (selection: FileGridSelection) => void
	onContextMenu?: (file: FileInfo, event: React.MouseEvent) => void
}) {
	const onItemSelect = useCallback(
		(file: FileInfo) => {
			onSelectionChange?.(new Set([file.id]))
		},
		[onSelectionChange],
	)

	const onItemContextMenu = useCallback(
		(file: FileInfo, event: React.MouseEvent) => {
			onContextMenu?.(file, event)
			onSelectionChange?.(new Set([file.id]))
		},
		[onContextMenu, onSelectionChange],
	)

	return (
		<div className="grid auto-cols-max grid-flow-col gap-3">
			{files.map((file) => (
				<FileGridItem
					selected={selectedFiles.has(file.id)}
					key={file.id}
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
	file: FileInfo
	onSelect?: (file: FileInfo) => void
	onContextMenu?: (file: FileInfo, event: React.MouseEvent) => void
}) {
	return (
		<button
			type="button"
			key={file.id}
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
