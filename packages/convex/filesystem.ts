import { v } from "convex/values"
import {
	apiKeyAuthenticatedQuery,
	authenticatedMutation,
	authenticatedQuery,
	authorizedGet,
} from "./functions"
import * as Directories from "./model/directories"
import * as Files from "./model/files"
import * as FileSystem from "./model/filesystem"
import {
	deleteItemsPermanently,
	emptyTrash as emptyTrashImpl,
	fetchFileUrl as fetchFileUrlImpl,
	restoreItems as restoreItemsImpl,
	VDirectoryHandle,
	VFileSystemHandle,
} from "./model/filesystem"
import * as Err from "./shared/error"
import type {
	DirectoryHandle,
	FileHandle,
	FileSystemHandle,
	FileSystemItem,
} from "./shared/filesystem"
import { FileType } from "./shared/filesystem"

export const moveItems = authenticatedMutation({
	args: {
		targetDirectory: VDirectoryHandle,
		items: v.array(VFileSystemHandle),
	},
	handler: async (ctx, { targetDirectory: targetDirectoryHandle, items }) => {
		const targetDirectory = await authorizedGet(
			ctx,
			targetDirectoryHandle.id,
		)
		if (!targetDirectory) {
			throw Err.create(
				Err.Code.DirectoryNotFound,
				`Directory ${targetDirectoryHandle.id} not found`,
			)
		}

		const directoryHandles: DirectoryHandle[] = []
		const fileHandles: FileHandle[] = []
		for (const item of items) {
			switch (item.kind) {
				case FileType.Directory:
					directoryHandles.push(item)
					break
				case FileType.File:
					fileHandles.push(item)
					break
			}
		}

		const [fileMoveResult, directoryMoveResult] = await Promise.all([
			Files.move(ctx, {
				targetDirectory: targetDirectoryHandle,
				items: fileHandles,
			}),
			Directories.move(ctx, {
				targetDirectory: targetDirectoryHandle,
				sourceDirectories: directoryHandles,
			}),
		])

		return {
			moved: [...directoryMoveResult.moved, ...fileMoveResult.moved],
			errors: [...fileMoveResult.errors, ...directoryMoveResult.errors],
		}
	},
})

export const moveToTrash = authenticatedMutation({
	args: {
		handles: v.array(VFileSystemHandle),
	},
	handler: async (ctx, { handles }) => {
		for (const handle of handles) {
			const item = await authorizedGet(ctx, handle.id)
			if (!item) {
				throw Err.create(
					Err.Code.NotFound,
					`Item ${handle.id} not found`,
				)
			}
		}

		// biome-ignore lint/suspicious/useIterableCallbackReturn: switch statement is exhaustive
		const promises = handles.map((handle) => {
			switch (handle.kind) {
				case FileType.File:
					return ctx.db
						.patch(handle.id, {
							deletedAt: Date.now(),
						})
						.then(() => handle)
				case FileType.Directory:
					return Directories.moveToTrashRecursive(ctx, handle).then(
						() => handle,
					)
			}
		})

		const results = await Promise.allSettled(promises)
		const errors: Err.ApplicationErrorData[] = []
		const okHandles: FileSystemHandle[] = []
		for (const result of results) {
			switch (result.status) {
				case "fulfilled":
					okHandles.push(result.value)
					break
				case "rejected":
					errors.push(Err.createJson(Err.Code.Internal))
					break
			}
		}

		return {
			deleted: okHandles,
			errors,
		}
	},
})

export const fetchDirectoryContent = authenticatedQuery({
	args: {
		directoryId: v.optional(v.id("directories")),
		trashed: v.boolean(),
	},
	handler: async (
		ctx,
		{ directoryId, trashed },
	): Promise<FileSystemItem[]> => {
		return await Directories.fetchContent(ctx, { directoryId, trashed })
	},
})

export const permanentlyDeleteItems = authenticatedMutation({
	args: {
		handles: v.array(VFileSystemHandle),
	},
	handler: async (ctx, { handles }) => {
		return await deleteItemsPermanently(ctx, { handles })
	},
})

export const emptyTrash = authenticatedMutation({
	handler: async (ctx) => {
		return await emptyTrashImpl(ctx)
	},
})

export const restoreItems = authenticatedMutation({
	args: {
		handles: v.array(VFileSystemHandle),
	},
	handler: async (ctx, { handles }) => {
		return await restoreItemsImpl(ctx, { handles })
	},
})

export const getStorageUrl = apiKeyAuthenticatedQuery({
	args: {
		storageId: v.id("_storage"),
	},
	handler: async (ctx, { storageId }) => {
		return await ctx.storage.getUrl(storageId)
	},
})

export const fetchFileUrl = authenticatedQuery({
	args: {
		fileId: v.id("files"),
	},
	handler: async (ctx, { fileId }) => {
		return await fetchFileUrlImpl(ctx, { fileId })
	},
})

export const openFile = authenticatedMutation({
	args: {
		fileId: v.id("files"),
	},
	handler: async (ctx, { fileId }) => {
		return await FileSystem.openFile(ctx, { fileId })
	},
})
