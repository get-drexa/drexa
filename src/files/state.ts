import { atom } from "jotai"
import type { Id } from "../../convex/_generated/dataModel"
import type {
	DirectoryItem,
	DirectoryItemKind,
} from "../../convex/model/directories"

export const contextMenuTargeItemAtom = atom<DirectoryItem | null>(null)
export const optimisticDeletedItemsAtom = atom(
	new Set<Id<"files"> | Id<"directories">>(),
)

export const newItemKindAtom = atom<DirectoryItemKind | null>(null)
