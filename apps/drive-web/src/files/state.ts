import type { Id } from "@fileone/convex/dataModel"
import type { DirectoryItem, DirectoryItemKind } from "@fileone/convex/types"
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
