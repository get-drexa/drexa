import type { Doc } from "@fileone/convex/_generated/dataModel"
import type { DirectoryInfo } from "@fileone/convex/model/directories"
import type { FileSystemItem } from "@fileone/convex/model/filesystem"
import { createContext } from "react"

type DirectoryPageContextType = {
	rootDirectory: Doc<"directories">
	directory: DirectoryInfo
	directoryContent: FileSystemItem[]
}

export const DirectoryPageContext = createContext<DirectoryPageContextType>(
	null as unknown as DirectoryPageContextType,
)
