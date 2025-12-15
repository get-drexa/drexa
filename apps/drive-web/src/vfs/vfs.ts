import { type } from "arktype"
import { Path } from "@/lib/path"

export const FileInfo = type({
	kind: "'file'",
	id: "string",
	name: "string",
	size: "number",
	mimeType: "string",
	createdAt: "string.date.iso.parse",
	updatedAt: "string.date.iso.parse",
	"deletedAt?": "string.date.iso.parse",
})
export type FileInfo = typeof FileInfo.infer

export const DirectoryInfo = type({
	kind: "'directory'",
	id: "string",
	name: "string",
	createdAt: "string.date.iso.parse",
	updatedAt: "string.date.iso.parse",
	"deletedAt?": "string.date.iso.parse",
})
export type DirectoryInfo = typeof DirectoryInfo.infer

export const DirectoryInfoWithPath = DirectoryInfo.and({
	path: Path,
})
export type DirectoryInfoWithPath = typeof DirectoryInfoWithPath.infer

export const DirectoryItem = type.or(DirectoryInfo, FileInfo)
export type DirectoryItem = typeof DirectoryItem.infer

export const DirectoryContent = DirectoryItem.array()
export type DirectoryContent = typeof DirectoryContent.infer
