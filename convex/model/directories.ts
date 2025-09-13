import type { Doc, Id } from "@convex/_generated/dataModel"
import type { MutationCtx, QueryCtx } from "@convex/_generated/server"

type Directory = {
	kind: "directory"
	doc: Doc<"directories">
}

type File = {
	kind: "file"
	doc: Doc<"files">
}

export type DirectoryItem = Directory | File

export async function fetchContent(
	ctx: QueryCtx,
	directoryId?: Id<"directories">,
): Promise<DirectoryItem[]> {
	const [files, directories] = await Promise.all([
		ctx.db
			.query("files")
			.withIndex("byDirectoryId", (q) => q.eq("directoryId", directoryId))
			.collect(),
		ctx.db
			.query("directories")
			.withIndex("byParentId", (q) => q.eq("parentId", directoryId))
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
	{ name, parentId }: { name: string; parentId?: Id<"directories"> },
): Promise<Id<"directories">> {
	const now = new Date().toISOString()
	return await ctx.db.insert("directories", {
		name,
		parentId,
		createdAt: now,
		updatedAt: now,
	})
}
