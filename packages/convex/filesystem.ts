import { v } from "convex/values"
import { authenticatedMutation } from "./functions"
import * as Directories from "./model/directories"
import * as Err from "./model/error"
import * as Files from "./model/files"
import type { DirectoryHandle, FileHandle } from "./model/filesystem"

const VDirectoryHandle = v.object({
	kind: v.literal("directory"),
	id: v.id("directories"),
})

const VFileHandle = v.object({
	kind: v.literal("file"),
	id: v.id("files"),
})

const VFileSystemHandle = v.union(VFileHandle, VDirectoryHandle)

export const moveItems = authenticatedMutation({
	args: {
		targetDirectory: VDirectoryHandle,
		items: v.array(VFileSystemHandle),
	},
	handler: async (ctx, { targetDirectory: targetDirectoryHandle, items }) => {
		const targetDirectory = await Directories.fetchHandle(
			ctx,
			targetDirectoryHandle,
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
				case "directory":
					directoryHandles.push(item)
					break
				case "file":
					fileHandles.push(item)
					break
			}
		}
		await Promise.all([
			Files.move(ctx, {
				targetDirectory: targetDirectoryHandle,
				items: fileHandles,
			}),
			Directories.move(ctx, {
				targetDirectory: targetDirectoryHandle,
				sourceDirectories: directoryHandles,
			}),
		])

		return { items, targetDirectory }
	},
})
