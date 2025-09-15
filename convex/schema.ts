import { defineSchema, defineTable } from "convex/server"
import { v } from "convex/values"

const schema = defineSchema({
	users: defineTable({
		jwtSubject: v.optional(v.string()), // JWT subject from identity provider (optional for migration)
	}).index("byJwtSubject", ["jwtSubject"]), // Unique index for JWT subject lookup
	files: defineTable({
		storageId: v.id("_storage"),
		userId: v.id("users"),
		directoryId: v.optional(v.id("directories")),
		name: v.string(),
		size: v.number(),
		mimeType: v.optional(v.string()),
		createdAt: v.string(),
		updatedAt: v.string(),
		deletedAt: v.optional(v.string()),
	})
		.index("byDirectoryId", ["directoryId", "deletedAt"])
		.index("byUserId", ["userId", "deletedAt"])
		.index("byDeletedAt", ["deletedAt"])
		.index("uniqueFileInDirectory", ["directoryId", "name", "deletedAt"]),
	directories: defineTable({
		name: v.string(),
		userId: v.id("users"),
		parentId: v.optional(v.id("directories")),
		createdAt: v.string(),
		updatedAt: v.string(),
		deletedAt: v.optional(v.string()),
	})
		.index("byUserId", ["userId", "deletedAt"])
		.index("byParentId", ["parentId", "deletedAt"])
		.index("uniqueDirectoryInDirectory", ["parentId", "name", "deletedAt"]),
})

export default schema
