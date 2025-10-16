import { v } from "convex/values"
import type { Doc, Id } from "../_generated/dataModel"
import type {
	AuthenticatedMutationCtx,
	AuthenticatedQueryCtx,
} from "../functions"
import { authorizedGet } from "../functions"
import * as Directories from "./directories"
import * as Err from "./error"
import * as Files from "./files"

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
export type DirectoryPath = [
	DirectoryPathComponent,
	...DirectoryPathComponent[],
]
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

export type DeleteResult = {
	deleted: {
		files: number
		directories: number
	}
	errors: Err.ApplicationErrorData[]
}

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

export async function queryRootDirectory(
	ctx: AuthenticatedQueryCtx,
): Promise<Doc<"directories"> | null> {
	return await ctx.db
		.query("directories")
		.withIndex("byParentId", (q) =>
			q.eq("userId", ctx.user._id).eq("parentId", undefined),
		)
		.first()
}

export async function ensureRootDirectory(
	ctx: AuthenticatedMutationCtx,
): Promise<Id<"directories">> {
	const existing = await queryRootDirectory(ctx)

	if (existing) {
		return existing._id
	}

	const now = Date.now()
	return await ctx.db.insert("directories", {
		name: "",
		createdAt: now,
		updatedAt: now,
		userId: ctx.user._id,
	})
}

/**
 * Recursively collects all file and directory handles from the given handles,
 * including all nested items. Only includes items that are in trash (deletedAt >= 0).
 */
async function collectAllHandlesRecursively(
	ctx: AuthenticatedMutationCtx,
	{ handles }: { handles: FileSystemHandle[] },
): Promise<{ fileHandles: FileHandle[]; directoryHandles: DirectoryHandle[] }> {
	const fileHandles: FileHandle[] = []
	const directoryHandles: DirectoryHandle[] = []

	// Process each handle to collect files and directories
	for (const handle of handles) {
		// Use a queue to process items iteratively instead of recursively
		const queue: FileSystemHandle[] = [handle]

		while (queue.length > 0) {
			const currentHandle = queue.shift()!

			// Add current item to appropriate collection
			if (currentHandle.kind === FileType.File) {
				fileHandles.push(currentHandle)
			} else {
				directoryHandles.push(currentHandle)
			}

			// If it's a directory, collect all children and add them to the queue
			if (currentHandle.kind === FileType.Directory) {
				// Get all child directories that are in trash (deletedAt >= 0)
				const childDirectories = await ctx.db
					.query("directories")
					.withIndex("byParentId", (q) =>
						q
							.eq("userId", ctx.user._id)
							.eq("parentId", currentHandle.id)
							.gte("deletedAt", 0),
					)
					.collect()

				// Get all child files that are in trash (deletedAt >= 0)
				const childFiles = await ctx.db
					.query("files")
					.withIndex("byDirectoryId", (q) =>
						q
							.eq("userId", ctx.user._id)
							.eq("directoryId", currentHandle.id)
							.gte("deletedAt", 0),
					)
					.collect()

				// Add child directories to queue for processing
				for (const childDir of childDirectories) {
					const childHandle = newDirectoryHandle(childDir._id)
					queue.push(childHandle)
				}

				// Add child files to file handles collection
				for (const childFile of childFiles) {
					const childFileHandle = newFileHandle(childFile._id)
					fileHandles.push(childFileHandle)
				}
			}
		}
	}

	return { fileHandles, directoryHandles }
}

/**
 * Restores deleted items by unsetting the deletedAt field recursively.
 * This includes all nested files and directories within the given handles.
 */
export async function restoreItems(
	ctx: AuthenticatedMutationCtx,
	{ handles }: { handles: FileSystemHandle[] },
) {
	// Collect all items to restore (including nested items)
	const { fileHandles, directoryHandles } =
		await collectAllHandlesRecursively(ctx, { handles })

	// Restore files and directories by unsetting deletedAt
	const [filesResult, directoriesResult] = await Promise.all([
		Files.restore(ctx, { items: fileHandles }),
		Directories.restore(ctx, { items: directoryHandles }),
	])

	// Combine results, handling null responses
	return {
		restored: {
			files: filesResult?.restored || 0,
			directories: directoriesResult?.restored || 0,
		},
		errors: [
			...(filesResult?.errors || []),
			...(directoriesResult?.errors || []),
		],
	}
}

export async function deleteItemsPermanently(
	ctx: AuthenticatedMutationCtx,
	{ handles }: { handles: FileSystemHandle[] },
): Promise<DeleteResult> {
	// Collect all items to delete (including nested items)
	const {
		fileHandles: fileHandlesToDelete,
		directoryHandles: directoryHandlesToDelete,
	} = await collectAllHandlesRecursively(ctx, { handles })

	// Delete files and directories using their respective models
	const [filesResult, directoriesResult] = await Promise.all([
		Files.deletePermanently(ctx, { items: fileHandlesToDelete }),
		Directories.deletePermanently(ctx, { items: directoryHandlesToDelete }),
	])

	// Combine results, handling null responses
	return {
		deleted: {
			files: filesResult?.deleted || 0,
			directories: directoriesResult?.deleted || 0,
		},
		errors: [
			...(filesResult?.errors || []),
			...(directoriesResult?.errors || []),
		],
	}
}

export async function emptyTrash(
	ctx: AuthenticatedMutationCtx,
): Promise<DeleteResult> {
	const rootDir = await queryRootDirectory(ctx)
	if (!rootDir) {
		throw Err.create(Err.Code.NotFound, "user root directory not found")
	}

	const dirs = await ctx.db
		.query("directories")
		.withIndex("byParentId", (q) =>
			q
				.eq("userId", ctx.user._id)
				.eq("parentId", rootDir._id)
				.gte("deletedAt", 0),
		)
		.collect()

	const files = await ctx.db
		.query("files")
		.withIndex("byDirectoryId", (q) =>
			q
				.eq("userId", ctx.user._id)
				.eq("directoryId", rootDir._id)
				.gte("deletedAt", 0),
		)
		.collect()

	if (dirs.length === 0 && files.length === 0) {
		return {
			deleted: {
				files: 0,
				directories: 0,
			},
			errors: [],
		}
	}

	return await deleteItemsPermanently(ctx, {
		handles: [
			...dirs.map((it) => newDirectoryHandle(it._id)),
			...files.map((it) => newFileHandle(it._id)),
		],
	})
}

export async function fetchFileUrl(
	ctx: AuthenticatedQueryCtx,
	{ fileId }: { fileId: Id<"files"> },
): Promise<string> {
	const file = await authorizedGet(ctx, fileId)
	if (!file) {
		throw Err.create(Err.Code.NotFound, "file not found")
	}

	const url = await ctx.storage.getUrl(file.storageId)
	if (!url) {
		throw Err.create(Err.Code.NotFound, "file not found")
	}

	return url
}
