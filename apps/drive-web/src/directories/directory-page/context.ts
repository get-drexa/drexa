import type { Doc } from "@fileone/convex/dataModel"
import type { FileSystemItem } from "@fileone/convex/filesystem"
import type { DirectoryInfo } from "@fileone/convex/types"
import { createContext } from "react"

type DirectoryPageContextType = {
	rootDirectory: Doc<"directories">
	directory: DirectoryInfo
	directoryContent: FileSystemItem[]
}

export const DirectoryPageContext = createContext<DirectoryPageContextType>(
	null as unknown as DirectoryPageContextType,
)
