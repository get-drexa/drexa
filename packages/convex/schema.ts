import { defineSchema, defineTable } from "convex/server"
import { v } from "convex/values"

const schema = defineSchema({
	files: defineTable({
		storageId: v.id("_storage"),
		userId: v.string(), // BetterAuth user IDs are strings, not Convex Ids
		directoryId: v.optional(v.id("directories")),
		name: v.string(),
		size: v.number(),
		mimeType: v.optional(v.string()),
		createdAt: v.number(),
		updatedAt: v.number(),
		deletedAt: v.optional(v.number()),
		lastAccessedAt: v.optional(v.number()),
	})
		.index("byDirectoryId", ["userId", "directoryId", "deletedAt"])
		.index("byUserId", ["userId", "deletedAt"])
		.index("byDeletedAt", ["deletedAt"])
		.index("byLastAccessedAt", ["userId", "lastAccessedAt"])
		.index("uniqueFileInDirectory", [
			"userId",
			"directoryId",
			"name",
			"deletedAt",
		]),
	directories: defineTable({
		name: v.string(),
		userId: v.string(), // BetterAuth user IDs are strings, not Convex Ids
		parentId: v.optional(v.id("directories")),
		createdAt: v.number(),
		updatedAt: v.number(),
		deletedAt: v.optional(v.number()),
	})
		.index("byUserId", ["userId", "deletedAt"])
		.index("byParentId", ["userId", "parentId", "deletedAt"])
		.index("uniqueDirectoryInDirectory", [
			"userId",
			"parentId",
			"name",
			"deletedAt",
		]),
})

export default schema
