import type { Doc, Id } from "@fileone/convex/_generated/dataModel"
import type {
	AuthenticatedMutationCtx,
	AuthenticatedQueryCtx,
} from "../functions"
import * as Err from "./error"
import {
	type DirectoryHandle,
	type FilePath,
	type FileSystemItem,
	FileType,
	newDirectoryHandle,
	type ReverseFilePath,
} from "./filesystem"

export type DirectoryInfo = Doc<"directories"> & { path: FilePath }

export async function fetchRoot(ctx: AuthenticatedQueryCtx) {
	return await ctx.db
		.query("directories")
		.withIndex("byParentId", (q) =>
			q.eq("userId", ctx.user._id).eq("parentId", undefined),
		)
		.first()
}

export async function fetchHandle(
	ctx: AuthenticatedQueryCtx,
	handle: DirectoryHandle,
): Promise<Doc<"directories">> {
	const directory = await ctx.db.get(handle.id)
	if (!directory || directory.userId !== ctx.user._id) {
		throw Err.create(
			Err.Code.DirectoryNotFound,
			`Directory ${handle.id} not found`,
		)
	}
	return directory
}

export async function fetch(
	ctx: AuthenticatedQueryCtx,
	{ directoryId }: { directoryId: Id<"directories"> },
): Promise<DirectoryInfo> {
	const directory = await ctx.db.get(directoryId)
	if (!directory) {
		throw Err.create(
			Err.Code.DirectoryNotFound,
			`Directory ${directoryId} not found`,
		)
	}

	const path: ReverseFilePath = [
		{
			handle: newDirectoryHandle(directoryId),
			name: directory.name,
		},
	]
	let parentDirId = directory.parentId
	while (parentDirId) {
		const parentDir = await ctx.db.get(parentDirId)
		if (parentDir) {
			path.push({
				handle: newDirectoryHandle(parentDir._id),
				name: parentDir.name,
			})
			parentDirId = parentDir.parentId
		} else {
			throw Err.create(Err.Code.Internal)
		}
	}

	return { ...directory, path: path.reverse() as FilePath }
}

export async function fetchContent(
	ctx: AuthenticatedQueryCtx,
	{ directoryId }: { directoryId?: Id<"directories"> } = {},
): Promise<FileSystemItem[]> {
	let dirId: Id<"directories"> | undefined
	if (directoryId) {
		dirId = directoryId
	}

	const [files, directories] = await Promise.all([
		ctx.db
			.query("files")
			.withIndex("byDirectoryId", (q) =>
				q
					.eq("userId", ctx.user._id)
					.eq("directoryId", dirId)
					.eq("deletedAt", undefined),
			)
			.collect(),
		ctx.db
			.query("directories")
			.withIndex("byParentId", (q) =>
				q
					.eq("userId", ctx.user._id)
					.eq("parentId", dirId)
					.eq("deletedAt", undefined),
			)
			.collect(),
	])

	const items: FileSystemItem[] = []
	for (const directory of directories) {
		items.push({ kind: FileType.Directory, doc: directory })
	}
	for (const file of files) {
		items.push({ kind: FileType.File, doc: file })
	}

	return items
}

export async function create(
	ctx: AuthenticatedMutationCtx,
	{ name, parentId }: { name: string; parentId: Id<"directories"> },
): Promise<Id<"directories">> {
	const parentDir = await ctx.db.get(parentId)
	if (!parentDir) {
		throw Err.create(
			Err.Code.DirectoryNotFound,
			`Parent directory ${parentId} not found`,
		)
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
	})
}

export async function move(
	ctx: AuthenticatedMutationCtx,
	{
		targetDirectory,
		sourceDirectories,
	}: {
		targetDirectory: DirectoryHandle
		sourceDirectories: DirectoryHandle[]
	},
) {
	const conflictCheckResults = await Promise.allSettled(
		sourceDirectories.map((directory) =>
			ctx.db.get(directory.id).then((d) => {
				if (!d) {
					throw Err.create(
						Err.Code.DirectoryNotFound,
						`Directory ${directory.id} not found`,
					)
				}
				return ctx.db
					.query("directories")
					.withIndex("uniqueDirectoryInDirectory", (q) =>
						q
							.eq("userId", ctx.user._id)
							.eq("parentId", targetDirectory.id)
							.eq("name", d.name)
							.eq("deletedAt", undefined),
					)
					.first()
			}),
		),
	)

	const errors: Err.ApplicationErrorData[] = []
	const okDirectories: DirectoryHandle[] = []
	conflictCheckResults.forEach((result, i) => {
		if (result.status === "fulfilled") {
			if (result.value) {
				errors.push(
					Err.createJson(
						Err.Code.Conflict,
						`Directory ${targetDirectory.id} already contains a directory with name ${result.value.name}`,
					),
				)
			} else {
				okDirectories.push(sourceDirectories[i]!)
			}
		} else if (result.status === "rejected") {
			errors.push(Err.createJson(Err.Code.Internal))
		}
	})

	const ignoredHandles = new Set<DirectoryHandle>()

	const promises: Promise<void>[] = []
	for (const handle of okDirectories) {
		if (handle.id === targetDirectory.id) {
			// if the directory that needs to be moved is the same as the dest directory
			// it is silently ignored
			ignoredHandles.add(handle)
		} else {
			promises.push(
				ctx.db.patch(handle.id, { parentId: targetDirectory.id }),
			)
		}
	}

	const results = await Promise.allSettled(promises)

	for (const updateResult of results) {
		if (updateResult.status === "rejected") {
			errors.push(Err.createJson(Err.Code.Internal))
		}
	}

	return {
		moved: okDirectories.filter((handle) => !ignoredHandles.has(handle)),
		errors,
	}
}

export async function moveToTrashRecursive(
	ctx: AuthenticatedMutationCtx,
	handle: DirectoryHandle,
): Promise<void> {
	const now = new Date().toISOString()

	const filesToDelete: Id<"files">[] = []
	const directoriesToDelete: Id<"directories">[] = []

	const directoryQueue: Id<"directories">[] = [handle.id]

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
