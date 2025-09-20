import type { Id } from "../_generated/dataModel"

export type DirectoryPathComponent = {
	handle: DirectoryHandle
	name: string
}

export type FilePathComponent = {
	handle: FileHandle
	name: string
}

export type PathComponent = FilePathComponent | DirectoryPathComponent

export type FilePath = [...DirectoryPathComponent[], PathComponent]

export type ReverseFilePath = [PathComponent, ...DirectoryPathComponent[]]

export type FileHandle = {
	kind: "file"
	id: Id<"files">
}

export type DirectoryHandle = {
	kind: "directory"
	id: Id<"directories">
}

export type FileSystemHandle = DirectoryHandle | FileHandle

export function newDirectoryHandle(id: Id<"directories">): DirectoryHandle {
	return { kind: "directory", id }
}

export function newFileHandle(id: Id<"files">): FileHandle {
	return { kind: "file", id }
}
