import type { Doc, Id } from "@fileone/convex/dataModel"
import type {
	AuthenticatedMutationCtx,
	AuthenticatedQueryCtx,
} from "../functions"
import { authorizedGet } from "../functions"
import type { ApplicationErrorData } from "../shared/error"
import { createErrorData, ErrorCode, error } from "../shared/error"
import {
	type DirectoryHandle,
	type DirectoryPath,
	type FileSystemItem,
	FileType,
	newDirectoryHandle,
} from "../shared/filesystem"

export type DirectoryInfo = Doc<"directories"> & { path: DirectoryPath }

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
	const directory = await authorizedGet(ctx, handle.id)
	if (!directory) {
		error({
			code: ErrorCode.NotFound,
			message: `Directory ${handle.id} not found`,
		})
	}
	return directory
}

export async function fetch(
	ctx: AuthenticatedQueryCtx,
	{ directoryId }: { directoryId: Id<"directories"> },
): Promise<DirectoryInfo> {
	const directory = await authorizedGet(ctx, directoryId)
	if (!directory) {
		error({
			code: ErrorCode.NotFound,
			message: `Directory ${directoryId} not found`,
		})
	}

	const path: DirectoryPath = [
		{
			handle: newDirectoryHandle(directoryId),
			name: directory.name,
		},
	]
	let parentDirId = directory.parentId
	while (parentDirId) {
		const parentDir = await authorizedGet(ctx, parentDirId)
		if (parentDir) {
			path.push({
				handle: newDirectoryHandle(parentDir._id),
				name: parentDir.name,
			})
			parentDirId = parentDir.parentId
		} else {
			error({
				code: ErrorCode.NotFound,
				message: "Parent directory not found",
			})
		}
	}

	return { ...directory, path: path.reverse() as DirectoryPath }
}

export async function fetchContent(
	ctx: AuthenticatedQueryCtx,
	{
		directoryId,
		trashed,
	}: { directoryId?: Id<"directories">; trashed: boolean },
): Promise<FileSystemItem[]> {
	let dirId: Id<"directories"> | undefined
	if (directoryId) {
		dirId = directoryId
	}

	const [files, directories] = await Promise.all([
		ctx.db
			.query("files")
			.withIndex("byDirectoryId", (q) => {
				if (trashed) {
					return q
						.eq("userId", ctx.user._id)
						.eq("directoryId", dirId)
						.gte("deletedAt", 0)
				}
				return q
					.eq("userId", ctx.user._id)
					.eq("directoryId", dirId)
					.eq("deletedAt", undefined)
			})
			.collect(),
		ctx.db
			.query("directories")
			.withIndex("byParentId", (q) => {
				if (trashed) {
					return q
						.eq("userId", ctx.user._id)
						.eq("parentId", dirId)
						.gte("deletedAt", 0)
				}
				return q
					.eq("userId", ctx.user._id)
					.eq("parentId", dirId)
					.eq("deletedAt", undefined)
			})
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
	const parentDir = await authorizedGet(ctx, parentId)
	if (!parentDir) {
		error({
			code: ErrorCode.NotFound,
			message: `Parent directory ${parentId} not found`,
		})
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
		error({
			code: ErrorCode.DirectoryExists,
			message: `Directory with name ${name} already exists in ${parentId ? `directory ${parentId}` : "root"}`,
		})
	}

	const now = Date.now()
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
			authorizedGet(ctx, directory.id).then((d) => {
				if (!d) {
					error({
						code: ErrorCode.NotFound,
						message: `Directory ${directory.id} not found`,
					})
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

	const errors: ApplicationErrorData[] = []
	const okDirectories: DirectoryHandle[] = []
	conflictCheckResults.forEach((result, i) => {
		if (result.status === "fulfilled") {
			if (result.value) {
				errors.push(
					createErrorData(
						ErrorCode.Conflict,
						`Directory ${targetDirectory.id} already contains a directory with name ${result.value.name}`,
					),
				)
			} else {
				okDirectories.push(sourceDirectories[i]!)
			}
		} else if (result.status === "rejected") {
			errors.push(createErrorData(ErrorCode.Internal))
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
				ctx.db.patch(handle.id, {
					parentId: targetDirectory.id,
					updatedAt: Date.now(),
				}),
			)
		}
	}

	const results = await Promise.allSettled(promises)

	for (const updateResult of results) {
		if (updateResult.status === "rejected") {
			errors.push(createErrorData(ErrorCode.Internal))
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
	const now = Date.now()

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

export async function deletePermanently(
	ctx: AuthenticatedMutationCtx,
	{
		items,
	}: {
		items: DirectoryHandle[]
	},
) {
	if (items.length === 0) {
		return null
	}

	const itemsToBeDeleted = await Promise.allSettled(
		items.map((item) => ctx.db.get(item.id)),
	).then((results) =>
		results.filter(
			(result): result is PromiseFulfilledResult<Doc<"directories">> =>
				result.status === "fulfilled" && result.value !== null,
		),
	)

	const deleteDirectoryPromises = itemsToBeDeleted.map((item) =>
		ctx.db.delete(item.value._id),
	)

	const deleteResults = await Promise.allSettled(deleteDirectoryPromises)

	const errors: ApplicationErrorData[] = []
	let successfulDeletions = 0
	for (const result of deleteResults) {
		if (result.status === "rejected") {
			errors.push(createErrorData(ErrorCode.Internal))
		} else {
			successfulDeletions += 1
		}
	}

	return { deleted: successfulDeletions, errors }
}

export async function restore(
	ctx: AuthenticatedMutationCtx,
	{
		items,
	}: {
		items: DirectoryHandle[]
	},
) {
	if (items.length === 0) {
		return null
	}

	const itemsToBeRestored = await Promise.allSettled(
		items.map((item) => ctx.db.get(item.id)),
	).then((results) =>
		results.filter(
			(result): result is PromiseFulfilledResult<Doc<"directories">> =>
				result.status === "fulfilled" && result.value !== null,
		),
	)

	const restoreDirectoryPromises = itemsToBeRestored.map((item) =>
		ctx.db.patch(item.value._id, {
			deletedAt: undefined,
			updatedAt: Date.now(),
		}),
	)

	const restoreResults = await Promise.allSettled(restoreDirectoryPromises)

	const errors: ApplicationErrorData[] = []
	let successfulRestorations = 0
	for (const result of restoreResults) {
		if (result.status === "rejected") {
			errors.push(createErrorData(ErrorCode.Internal))
		} else {
			successfulRestorations += 1
		}
	}

	return { restored: successfulRestorations, errors }
}
