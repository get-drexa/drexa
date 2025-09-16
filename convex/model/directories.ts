import type { Doc, Id } from "@convex/_generated/dataModel"
import type {
	AuthenticatedMutationCtx,
	AuthenticatedQueryCtx,
} from "../functions"
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
	ctx: AuthenticatedQueryCtx,
	directoryId?: Id<"directories">,
): Promise<DirectoryItem[]> {
	const [files, directories] = await Promise.all([
		ctx.db
			.query("files")
			.withIndex("byDirectoryId", (q) =>
				q
					.eq("userId", ctx.user._id)
					.eq("directoryId", directoryId)
					.eq("deletedAt", undefined),
			)
			.collect(),
		ctx.db
			.query("directories")
			.withIndex("byParentId", (q) =>
				q
					.eq("userId", ctx.user._id)
					.eq("parentId", directoryId)
					.eq("deletedAt", undefined),
			)
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
	ctx: AuthenticatedMutationCtx,
	{ name, parentId }: { name: string; parentId?: Id<"directories"> },
): Promise<Id<"directories">> {
	let parentDir: Doc<"directories"> | null = null
	if (parentId) {
		parentDir = await ctx.db.get(parentId)
		if (!parentDir) {
			throw Err.create(
				Err.Code.DirectoryNotFound,
				`Parent directory ${parentId} not found`,
			)
		}
	}

	const existing = await ctx.db
		.query("directories")
		.withIndex("uniqueDirectoryInDirectory", (q) =>
			q
				.eq("userId", ctx.user._id)
				.eq("parentId", parentId)
				.eq("name", name)
				.eq("deletedAt", undefined),
		)
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
		userId: ctx.user._id,
		createdAt: now,
		updatedAt: now,
		path: parentDir ? `${parentDir.path}/${name}` : "/",
	})
}

export async function moveToTrashRecursive(
	ctx: AuthenticatedMutationCtx,
	directoryId: Id<"directories">,
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
					.eq("userId", ctx.user._id)
					.eq("directoryId", currentDirectoryId)
					.eq("deletedAt", undefined),
			)
			.collect()

		for (const file of files) {
			filesToDelete.push(file._id)
		}

		const subdirectories = await ctx.db
			.query("directories")
			.withIndex("byParentId", (q) =>
				q
					.eq("userId", ctx.user._id)
					.eq("parentId", currentDirectoryId)
					.eq("deletedAt", undefined),
			)
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
