import { ConvexError, v } from "convex/values"
import type { Doc, Id } from "../_generated/dataModel"
import {
	type AuthenticatedMutationCtx,
	type AuthenticatedQueryCtx,
	authorizedGet,
} from "../functions"
import { ErrorCode, error } from "../shared/error"
import type {
	DirectoryHandle,
	FileHandle,
	FileSystemHandle,
} from "../shared/filesystem"
import {
	FileType,
	newDirectoryHandle,
	newFileHandle,
} from "../shared/filesystem"
import * as Directories from "./directories"
import * as FilePreview from "./filepreview"
import * as Files from "./files"
import * as FileShare from "./fileshare"
import * as User from "./user"

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
	ctx: AuthenticatedQueryCtx | AuthenticatedMutationCtx,
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

	for (const handle of handles) {
		const queue: FileSystemHandle[] = [handle]

		while (queue.length > 0) {
			const currentHandle = queue.shift()!

			if (currentHandle.kind === FileType.File) {
				fileHandles.push(currentHandle)
			} else {
				directoryHandles.push(currentHandle)
			}

			if (currentHandle.kind === FileType.Directory) {
				const childDirectories = await ctx.db
					.query("directories")
					.withIndex("byParentId", (q) =>
						q
							.eq("userId", ctx.user._id)
							.eq("parentId", currentHandle.id)
							.gte("deletedAt", 0),
					)
					.collect()

				const childFiles = await ctx.db
					.query("files")
					.withIndex("byDirectoryId", (q) =>
						q
							.eq("userId", ctx.user._id)
							.eq("directoryId", currentHandle.id)
							.gte("deletedAt", 0),
					)
					.collect()

				for (const childDir of childDirectories) {
					queue.push(newDirectoryHandle(childDir._id))
				}

				for (const childFile of childFiles) {
					fileHandles.push(newFileHandle(childFile._id))
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
	const { fileHandles, directoryHandles } =
		await collectAllHandlesRecursively(ctx, { handles })

	const [filesResult, directoriesResult] = await Promise.all([
		Files.restore(ctx, { items: fileHandles }),
		Directories.restore(ctx, { items: directoryHandles }),
	])

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
) {
	const { fileHandles, directoryHandles } =
		await collectAllHandlesRecursively(ctx, { handles })

	const [filesResult, directoriesResult] = await Promise.all([
		Files.deletePermanently(ctx, { items: fileHandles }),
		Directories.deletePermanently(ctx, { items: directoryHandles }),
	])

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

export async function emptyTrash(ctx: AuthenticatedMutationCtx) {
	const rootDir = await queryRootDirectory(ctx)
	if (!rootDir) {
		error({
			code: ErrorCode.NotFound,
			message: "user root directory not found",
		})
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
		error({
			code: ErrorCode.NotFound,
			message: "file not found",
		})
	}

	const url = await ctx.storage.getUrl(file.storageId)
	if (!url) {
		error({
			code: ErrorCode.NotFound,
			message: "file not found",
		})
	}

	return url
}

export async function openFile(
	ctx: AuthenticatedMutationCtx,
	{ fileId }: { fileId: Id<"files"> },
) {
	const file = await authorizedGet(ctx, fileId)
	if (!file) {
		error({
			code: ErrorCode.NotFound,
			message: "file not found",
		})
	}

	const fileShare = await FilePreview.find(ctx, {
		storageId: file.storageId,
	})
	if (fileShare && !FileShare.hasExpired(fileShare)) {
		await FilePreview.extend(ctx, { doc: fileShare })
		return {
			file,
			shareToken: fileShare.shareToken,
		}
	}

	const [newFileShare] = await Promise.all([
		FilePreview.create(ctx, {
			storageId: file.storageId,
		}),
		ctx.db.patch(fileId, {
			lastAccessedAt: Date.now(),
		}),
	])

	return {
		file,
		shareToken: newFileShare.shareToken,
	}
}

export async function saveFile(
	ctx: AuthenticatedMutationCtx,
	{
		name,
		storageId,
		directoryId,
	}: {
		name: string
		storageId: Id<"_storage">
		directoryId: Id<"directories">
	},
) {
	const directory = await authorizedGet(ctx, directoryId)
	if (!directory) {
		error({
			code: ErrorCode.NotFound,
			message: "directory not found",
		})
	}

	const [fileMetadata, userInfo] = await Promise.all([
		ctx.db.system.get(storageId),
		User.queryInfo(ctx),
	])
	if (!fileMetadata || !userInfo) {
		throw new ConvexError({ message: "Internal server error" })
	}

	if (
		userInfo.storageUsageBytes + fileMetadata.size >
		userInfo.storageQuotaBytes
	) {
		await ctx.storage.delete(storageId)
		error({
			code: ErrorCode.StorageQuotaExceeded,
			message: "Storage quota exceeded",
		})
	}

	const now = Date.now()

	const [fileId] = await Promise.all([
		ctx.db.insert("files", {
			name,
			userId: ctx.user._id,
			createdAt: now,
			updatedAt: now,
			storageId,
			directoryId,
			size: fileMetadata.size,
			mimeType: fileMetadata.contentType,
		}),
		ctx.db.patch(userInfo._id, {
			storageUsageBytes: userInfo.storageUsageBytes + fileMetadata.size,
		}),
	])

	return fileId
}

export async function fetchRecentFiles(
	ctx: AuthenticatedQueryCtx,
	{ limit }: { limit: number },
) {
	return await ctx.db
		.query("files")
		.withIndex("byLastAccessedAt", (q) =>
			q
				.eq("userId", ctx.user._id)
				.eq("deletedAt", undefined)
				.gte("lastAccessedAt", 0),
		)
		.order("desc")
		.take(limit)
}
