import { v } from "convex/values"
import type { Id } from "./_generated/dataModel"
import { authenticatedMutation, authenticatedQuery } from "./functions"
import type { DirectoryItem } from "./model/directories"
import * as Directories from "./model/directories"

export const generateUploadUrl = authenticatedMutation({
	handler: async (ctx) => {
		// ctx.user and ctx.identity are automatically available
		return await ctx.storage.generateUploadUrl()
	},
})

export const fetchFiles = authenticatedQuery({
	args: {
		directoryId: v.optional(v.id("directories")),
	},
	handler: async (ctx, { directoryId }) => {
		return await ctx.db
			.query("files")
			.withIndex("byDirectoryId", (q) => q.eq("directoryId", directoryId))
			.filter((q) => q.eq(q.field("userId"), ctx.user._id))
			.collect()
	},
})

export const fetchDirectoryContent = authenticatedQuery({
	args: {
		directoryId: v.optional(v.id("directories")),
	},
	handler: async (ctx, { directoryId }): Promise<DirectoryItem[]> => {
		return await Directories.fetchContent(ctx, directoryId, ctx.user._id)
	},
})

export const createDirectory = authenticatedMutation({
	args: {
		name: v.string(),
		directoryId: v.optional(v.id("directories")),
	},
	handler: async (ctx, { name, directoryId }): Promise<Id<"directories">> => {
		return await Directories.create(ctx, {
			name,
			parentId: directoryId,
			userId: ctx.user._id,
		})
	},
})

export const saveFile = authenticatedMutation({
	args: {
		name: v.string(),
		size: v.number(),
		directoryId: v.optional(v.id("directories")),
		storageId: v.id("_storage"),
		mimeType: v.optional(v.string()),
	},
	handler: async (ctx, { name, storageId, directoryId, size, mimeType }) => {
		const now = new Date().toISOString()

		await ctx.db.insert("files", {
			name,
			size,
			storageId,
			directoryId,
			userId: ctx.user._id,
			mimeType,
			createdAt: now,
			updatedAt: now,
		})
	},
})

export const moveToTrash = authenticatedMutation({
	args: {
		kind: v.union(v.literal("file"), v.literal("directory")),
		itemId: v.union(v.id("files"), v.id("directories")),
	},
	handler: async (ctx, { itemId, kind }) => {
		// Verify ownership before allowing deletion
		switch (kind) {
			case "file": {
				const file = await ctx.db.get(itemId as Id<"files">)
				if (!file || file.userId !== ctx.user._id) {
					throw new Error("File not found or access denied")
				}
				await ctx.db.patch(itemId, {
					deletedAt: new Date().toISOString(),
				})
				break
			}
			case "directory": {
				const directory = await ctx.db.get(itemId as Id<"directories">)
				if (!directory || directory.userId !== ctx.user._id) {
					throw new Error("Directory not found or access denied")
				}
				await Directories.moveToTrashRecursive(
					ctx,
					itemId as Id<"directories">,
					ctx.user._id,
				)
				break
			}
		}

		return itemId
	},
})
