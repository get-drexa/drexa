import type { Doc } from "@fileone/convex/_generated/dataModel"
import type { DirectoryItem } from "@fileone/convex/model/directories"
import { createContext } from "react"

type DirectoryPageContextType = {
	directory: Doc<"directories">
	directoryContent: DirectoryItem[]
}

export const DirectoryPageContext = createContext<DirectoryPageContextType>(
	null as unknown as DirectoryPageContextType,
)
