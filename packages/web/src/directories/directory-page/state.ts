import type { Doc, Id } from "@fileone/convex/_generated/dataModel"
import type { DirectoryItemKind } from "@fileone/convex/model/directories"
import type { FileSystemItem, FileType } from "@fileone/convex/model/filesystem"
import type { RowSelectionState } from "@tanstack/react-table"
import { atom } from "jotai"
import type { FileDragInfo } from "../../files/use-file-drop"

export const contextMenuTargeItemsAtom = atom<FileSystemItem[]>([])
export const optimisticDeletedItemsAtom = atom(
	new Set<Id<"files"> | Id<"directories">>(),
)

export const selectedFileRowsAtom = atom<RowSelectionState>({})

export const newFileTypeAtom = atom<FileType | null>(null)

export const itemBeingRenamedAtom = atom<{
	originalItem: FileSystemItem
	name: string
} | null>(null)

export const openedFileAtom = atom<Doc<"files"> | null>(null)

export const dragInfoAtom = atom<FileDragInfo | null>(null)
