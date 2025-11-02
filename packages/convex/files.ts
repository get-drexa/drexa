import type { Id } from "@fileone/convex/dataModel"
import { v } from "convex/values"
import {
	authenticatedMutation,
	authenticatedQuery,
	authorizedGet,
} from "./functions"
import * as Directories from "./model/directories"
import * as Files from "./model/files"
import * as User from "./model/user"
import * as Err from "./shared/error"

export const generateUploadUrl = authenticatedMutation({
	handler: async (ctx) => {
		const usageStatistics = await User.queryCachedUsageStatistics(ctx)
		if (!usageStatistics) {
			throw Err.create(Err.Code.Internal, "Internal server error")
		}
		if (
			usageStatistics.storageUsageBytes >=
			usageStatistics.storageQuotaBytes
		) {
			throw Err.create(Err.Code.Forbidden, "Storage quota exceeded")
		}
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
			.withIndex("byDirectoryId", (q) =>
				q.eq("userId", ctx.user._id).eq("directoryId", directoryId),
			)
			.collect()
	},
})

export const fetchRootDirectory = authenticatedQuery({
	handler: async (ctx) => {
		return await Directories.fetchRoot(ctx)
	},
})

export const fetchDirectory = authenticatedQuery({
	args: {
		directoryId: v.id("directories"),
	},
	handler: async (ctx, { directoryId }) => {
		const directory = await authorizedGet(ctx, directoryId)
		if (!directory) {
			throw new Error("Directory not found")
		}
		return await Directories.fetch(ctx, { directoryId })
	},
})

export const createDirectory = authenticatedMutation({
	args: {
		name: v.string(),
		directoryId: v.id("directories"),
	},
	handler: async (ctx, { name, directoryId }): Promise<Id<"directories">> => {
		const parentDirectory = await authorizedGet(ctx, directoryId)
		if (!parentDirectory) {
			throw new Error("Parent directory not found")
		}

		return await Directories.create(ctx, {
			name,
			parentId: directoryId,
		})
	},
})

export const saveFile = authenticatedMutation({
	args: {
		name: v.string(),
		directoryId: v.id("directories"),
		storageId: v.id("_storage"),
	},
	handler: async (ctx, { name, storageId, directoryId }) => {
		const directory = await authorizedGet(ctx, directoryId)
		if (!directory) {
			throw new Error("Directory not found")
		}

		const now = Date.now()

		const fileMetadata = await Promise.all([
			ctx.db.system.get(storageId),
			ctx.user.queryUsageStatistics(),
		])
		if (!fileMetadata) {
			throw Err.create(Err.Code.Internal, "Internal server error")
		}

		await Promise.all([
			ctx.db.insert("files", {
				name,
				size: fileMetadata.size,
				storageId,
				directoryId,
				userId: ctx.user._id,
				mimeType,
				createdAt: now,
				updatedAt: now,
			}),
		])
	},
})

export const renameFile = authenticatedMutation({
	args: {
		directoryId: v.optional(v.id("directories")),
		itemId: v.id("files"),
		newName: v.string(),
	},
	handler: async (ctx, { directoryId, itemId, newName }) => {
		const file = await authorizedGet(ctx, itemId)
		if (!file) {
			throw new Error("File not found")
		}

		await Files.renameFile(ctx, { directoryId, itemId, newName })
	},
})
