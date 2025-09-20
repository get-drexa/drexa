import type { Id } from "../_generated/dataModel"

export type DirectoryPathComponent = {
	id: Id<"directories">
	name: string
}

export type FilePathComponent = {
	id: Id<"files">
	name: string
}

export type PathComponent = FilePathComponent | DirectoryPathComponent

export type FilePath = [...DirectoryPathComponent[], PathComponent]

export type ReverseFilePath = [PathComponent, ...DirectoryPathComponent[]]
