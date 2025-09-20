import type { Doc, Id } from "@fileone/convex/_generated/dataModel"
import type {
	DirectoryItem,
	DirectoryItemKind,
} from "@fileone/convex/model/directories"
import type { RowSelectionState } from "@tanstack/react-table"
import { atom } from "jotai"

export const contextMenuTargeItemAtom = atom<DirectoryItem | null>(null)
export const optimisticDeletedItemsAtom = atom(
	new Set<Id<"files"> | Id<"directories">>(),
)

export const selectedFileRowsAtom = atom<RowSelectionState>({})

export const newItemKindAtom = atom<DirectoryItemKind | null>(null)

export const itemBeingRenamedAtom = atom<{
	kind: DirectoryItemKind
	originalItem: DirectoryItem
	name: string
} | null>(null)

export const openedFileAtom = atom<Doc<"files"> | null>(null)

export const dragInfoAtom = atom<{
	source: DirectoryItem
	items: Id<"files">[]
} | null>(null)
