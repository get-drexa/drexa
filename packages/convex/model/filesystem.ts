import { v } from "convex/values"
import type { Doc, Id } from "../_generated/dataModel"

export enum FileType {
	File = "File",
	Directory = "Directory",
}

export type Directory = {
	kind: FileType.Directory
	doc: Doc<"directories">
}
export type File = {
	kind: FileType.File
	doc: Doc<"files">
}
export type FileSystemItem = Directory | File

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

export type DirectoryHandle = {
	kind: FileType.Directory
	id: Id<"directories">
}
export type FileHandle = {
	kind: FileType.File
	id: Id<"files">
}
export type FileSystemHandle = DirectoryHandle | FileHandle

export function newFileSystemHandle(item: FileSystemItem): FileSystemHandle {
	console.log("item", item)
	switch (item.kind) {
		case FileType.File:
			return { kind: item.kind, id: item.doc._id }
		case FileType.Directory:
			return { kind: item.kind, id: item.doc._id }
	}
}

export function isSameHandle(
	handle1: FileSystemHandle,
	handle2: FileSystemHandle,
): boolean {
	return handle1.kind === handle2.kind && handle1.id === handle2.id
}

export function newDirectoryHandle(id: Id<"directories">): DirectoryHandle {
	return { kind: FileType.Directory, id }
}

export function newFileHandle(id: Id<"files">): FileHandle {
	return { kind: FileType.File, id }
}

export const VDirectoryHandle = v.object({
	kind: v.literal(FileType.Directory),
	id: v.id("directories"),
})
export const VFileHandle = v.object({
	kind: v.literal(FileType.File),
	id: v.id("files"),
})
export const VFileSystemHandle = v.union(VFileHandle, VDirectoryHandle)
