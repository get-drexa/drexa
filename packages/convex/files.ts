import type { Id } from "@fileone/convex/dataModel"
import { ConvexError, v } from "convex/values"
import {
	authenticatedMutation,
	authenticatedQuery,
	authorizedGet,
} from "./functions"
import * as Directories from "./model/directories"
import * as Files from "./model/files"
import * as User from "./model/user"
import { ErrorCode, error } from "./shared/error"

export const generateUploadUrl = authenticatedMutation({
	handler: async (ctx) => {
		const userInfo = await User.queryInfo(ctx)
		if (!userInfo) {
			throw new ConvexError({ message: "Internal server error" })
		}
		if (userInfo.storageUsageBytes >= userInfo.storageQuotaBytes) {
			throw new ConvexError({
				code: ErrorCode.StorageQuotaExceeded,
				message: "Storage quota exceeded",
			})
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
			error({
				code: ErrorCode.NotFound,
				message: "Directory not found",
			})
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
			error({
				code: ErrorCode.NotFound,
				message: "Parent directory not found",
			})
		}

		return await Directories.create(ctx, {
			name,
			parentId: directoryId,
		})
	},
})
