import { v } from "convex/values"
import { mutation } from "../_generated/server"

export const moveToTrash = mutation({
	args: {
		fileId: v.id("files"),
	},
	handler: async (ctx, { fileId }) => {
		await ctx.db.patch(fileId, {
			deletedAt: new Date().toISOString(),
		})
	},
})
