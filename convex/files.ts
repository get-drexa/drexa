import { v } from "convex/values"
import type { Id } from "./_generated/dataModel"
import { mutation, query } from "./_generated/server"
import type { DirectoryItem } from "./model/directories"
import * as Directories from "./model/directories"

export const generateUploadUrl = mutation({
	handler: async (ctx) => {
		return await ctx.storage.generateUploadUrl()
	},
})

export const fetchFiles = query({
	args: {
		directoryId: v.optional(v.id("directories")),
	},
	handler: async (ctx, { directoryId }) => {
		return await ctx.db
			.query("files")
			.withIndex("byDirectoryId", (q) => q.eq("directoryId", directoryId))
			.collect()
	},
})

export const fetchDirectoryContent = query({
	args: {
		directoryId: v.optional(v.id("directories")),
	},
	handler: async (ctx, { directoryId }): Promise<DirectoryItem[]> => {
		return await Directories.fetchContent(ctx, directoryId)
	},
})

export const createDirectory = mutation({
	args: {
		name: v.string(),
		directoryId: v.optional(v.id("directories")),
	},
	handler: async (ctx, { name, directoryId }): Promise<Id<"directories">> => {
		return await Directories.create(ctx, { name, parentId: directoryId })
	},
})

export const saveFile = mutation({
	args: {
		name: v.string(),
		size: v.number(),
		directoryId: v.optional(v.id("directories")),
		storageId: v.id("_storage"),
		userId: v.id("users"),
	},
	handler: async (ctx, { name, storageId, directoryId, userId, size }) => {
		const now = new Date().toISOString()
		await ctx.db.insert("files", {
			name,
			size,
			storageId,
			directoryId,
			userId,
			createdAt: now,
			updatedAt: now,
		})
	},
})

export const moveToTrash = mutation({
	args: {
		kind: v.union(v.literal("file"), v.literal("directory")),
		itemId: v.union(v.id("files"), v.id("directories")),
	},
	handler: async (ctx, { itemId, kind }) => {
		switch (kind) {
			case "file":
				await ctx.db.patch(itemId, {
					deletedAt: new Date().toISOString(),
				})
				break
			case "directory":
				await Directories.moveToTrashRecursive(
					ctx,
					itemId as Id<"directories">,
				)
				break
		}

		return itemId
	},
})
