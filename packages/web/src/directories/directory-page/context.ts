import type { Doc } from "@fileone/convex/_generated/dataModel"
import type {
	DirectoryInfo,
	DirectoryItem,
} from "@fileone/convex/model/directories"
import { createContext } from "react"

type DirectoryPageContextType = {
	rootDirectory: Doc<"directories">
	directory: DirectoryInfo
	directoryContent: DirectoryItem[]
}

export const DirectoryPageContext = createContext<DirectoryPageContextType>(
	null as unknown as DirectoryPageContextType,
)
