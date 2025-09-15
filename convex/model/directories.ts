import type { Doc, Id } from "@convex/_generated/dataModel"
import type { MutationCtx, QueryCtx } from "@convex/_generated/server"
import * as Err from "./error"

type Directory = {
	kind: "directory"
	doc: Doc<"directories">
}

type File = {
	kind: "file"
	doc: Doc<"files">
}

export type DirectoryItem = Directory | File
export type DirectoryItemKind = DirectoryItem["kind"]

export async function fetchContent(
	ctx: QueryCtx,
	directoryId?: Id<"directories">,
	userId?: Id<"users">,
): Promise<DirectoryItem[]> {
	const [files, directories] = await Promise.all([
		ctx.db
			.query("files")
			.withIndex("byDirectoryId", (q) =>
				q.eq("directoryId", directoryId).eq("deletedAt", undefined),
			)
			.filter((q) => userId ? q.eq(q.field("userId"), userId) : q.neq(q.field("userId"), null))
			.collect(),
		ctx.db
			.query("directories")
			.withIndex("byParentId", (q) => q.eq("parentId", directoryId))
			.filter((q) => q.eq(q.field("deletedAt"), undefined))
			.filter((q) => userId ? q.eq(q.field("userId"), userId) : q.neq(q.field("userId"), null))
			.collect(),
	])

	const items: DirectoryItem[] = []
	for (const directory of directories) {
		items.push({ kind: "directory", doc: directory })
	}
	for (const file of files) {
		items.push({ kind: "file", doc: file })
	}

	return items
}

export async function create(
	ctx: MutationCtx,
	{ name, parentId, userId }: { name: string; parentId?: Id<"directories">; userId: Id<"users"> },
): Promise<Id<"directories">> {
	// Check if parent directory exists and belongs to user
	if (parentId) {
		const parentDir = await ctx.db.get(parentId)
		if (!parentDir || parentDir.userId !== userId) {
			throw Err.create(
				Err.Code.DirectoryExists,
				"Parent directory not found or access denied",
			)
		}
	}

	const existing = await ctx.db
		.query("directories")
		.withIndex("uniqueDirectoryInDirectory", (q) =>
			q.eq("parentId", parentId).eq("name", name),
		)
		.filter((q) => q.eq(q.field("userId"), userId))
		.first()

	if (existing) {
		throw Err.create(
			Err.Code.DirectoryExists,
			`Directory with name ${name} already exists in ${parentId ? `directory ${parentId}` : "root"}`,
		)
	}

	const now = new Date().toISOString()
	return await ctx.db.insert("directories", {
		name,
		parentId,
		userId,
		createdAt: now,
		updatedAt: now,
	})
}

export async function moveToTrashRecursive(
	ctx: MutationCtx,
	directoryId: Id<"directories">,
	userId: Id<"users">,
): Promise<void> {
	const now = new Date().toISOString()

	const filesToDelete: Id<"files">[] = []
	const directoriesToDelete: Id<"directories">[] = []

	const directoryQueue: Id<"directories">[] = [directoryId]

	while (directoryQueue.length > 0) {
		const currentDirectoryId = directoryQueue.shift()!
		directoriesToDelete.push(currentDirectoryId)

		const files = await ctx.db
			.query("files")
			.withIndex("byDirectoryId", (q) =>
				q
					.eq("directoryId", currentDirectoryId)
					.eq("deletedAt", undefined),
			)
			.filter((q) => q.eq(q.field("userId"), userId))
			.collect()

		for (const file of files) {
			filesToDelete.push(file._id)
		}

		const subdirectories = await ctx.db
			.query("directories")
			.withIndex("byParentId", (q) =>
				q.eq("parentId", currentDirectoryId).eq("deletedAt", undefined),
			)
			.filter((q) => q.eq(q.field("userId"), userId))
			.collect()

		for (const subdirectory of subdirectories) {
			directoryQueue.push(subdirectory._id)
		}
	}

	const filePatches = filesToDelete.map((fileId) =>
		ctx.db.patch(fileId, { deletedAt: now }),
	)

	const directoryPatches = directoriesToDelete.map((dirId) =>
		ctx.db.patch(dirId, { deletedAt: now }),
	)

	await Promise.all([...filePatches, ...directoryPatches])
}
