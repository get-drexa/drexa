import { createContext } from "react"
import type { DirectoryContent, DirectoryInfoWithPath } from "@/vfs/vfs"

type DirectoryPageContextType = {
	directory: DirectoryInfoWithPath
	directoryContent: DirectoryContent
}

export const DirectoryPageContext = createContext<DirectoryPageContextType>(
	null as unknown as DirectoryPageContextType,
)
