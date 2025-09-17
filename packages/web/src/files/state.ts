import type { Id } from "@fileone/convex/_generated/dataModel"
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
