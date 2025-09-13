import { defineSchema, defineTable } from "convex/server"
import { v } from "convex/values"

const schema = defineSchema({
	users: defineTable({
		name: v.string(),
		email: v.string(),
		createdAt: v.string(),
		updatedAt: v.string(),
	}),
	files: defineTable({
		storageId: v.id("_storage"),
		userId: v.id("users"),
		directoryId: v.optional(v.id("directories")),
		name: v.string(),
		size: v.number(),
		createdAt: v.string(),
		updatedAt: v.string(),
	}).index("byDirectoryId", ["directoryId"]),
	directories: defineTable({
		name: v.string(),
		parentId: v.optional(v.id("directories")),
		createdAt: v.string(),
		updatedAt: v.string(),
	}).index("byParentId", ["parentId"]),
})

export default schema
