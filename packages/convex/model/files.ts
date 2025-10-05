import type { Doc, Id } from "../_generated/dataModel"
import type { AuthenticatedMutationCtx } from "../functions"
import * as Err from "./error"
import type { DirectoryHandle, FileHandle } from "./filesystem"

export async function renameFile(
	ctx: AuthenticatedMutationCtx,
	{
		directoryId,
		itemId,
		newName,
	}: {
		directoryId?: Id<"directories">
		itemId: Id<"files">
		newName: string
	},
) {
	const existing = await ctx.db
		.query("files")
		.withIndex("uniqueFileInDirectory", (q) =>
			q
				.eq("userId", ctx.user._id)
				.eq("directoryId", directoryId)
				.eq("name", newName)
				.eq("deletedAt", undefined),
		)
		.first()

	if (existing) {
		throw Err.create(
			Err.Code.FileExists,
			`File with name ${newName} already exists in ${directoryId ? `directory ${directoryId}` : "root"}`,
		)
	}

	await ctx.db.patch(itemId, { name: newName, updatedAt: Date.now() })
}

export async function move(
	ctx: AuthenticatedMutationCtx,
	{
		targetDirectory: targetDirectoryHandle,
		items,
	}: {
		targetDirectory: DirectoryHandle
		items: FileHandle[]
	},
) {
	const conflictCheckResults = await Promise.allSettled(
		items.map((fileHandle) =>
			ctx.db.get(fileHandle.id).then((f) => {
				if (!f) {
					throw Err.create(
						Err.Code.FileNotFound,
						`File ${fileHandle.id} not found`,
					)
				}
				return ctx.db
					.query("files")
					.withIndex("uniqueFileInDirectory", (q) =>
						q
							.eq("userId", ctx.user._id)
							.eq("directoryId", targetDirectoryHandle.id)
							.eq("name", f.name)
							.eq("deletedAt", undefined),
					)
					.first()
			}),
		),
	)

	const errors: Err.ApplicationErrorData[] = []
	const okFiles: FileHandle[] = []
	conflictCheckResults.forEach((result, i) => {
		if (result.status === "fulfilled") {
			if (result.value) {
				errors.push(
					Err.createJson(
						Err.Code.Conflict,
						`Directory ${targetDirectoryHandle.id} already contains a file with name ${result.value.name}`,
					),
				)
			} else {
				okFiles.push(items[i])
			}
		} else if (result.status === "rejected") {
			errors.push(Err.createJson(Err.Code.Internal))
		}
	})

	const results = await Promise.allSettled(
		okFiles.map((handle) =>
			ctx.db.patch(handle.id, {
				directoryId: targetDirectoryHandle.id,
				updatedAt: Date.now(),
			}),
		),
	)

	for (const updateResult of results) {
		if (updateResult.status === "rejected") {
			errors.push(Err.createJson(Err.Code.Internal))
		}
	}

	return { moved: okFiles, errors }
}

export async function deletePermanently(
	ctx: AuthenticatedMutationCtx,
	{
		items,
	}: {
		items: FileHandle[]
	},
) {
	if (items.length === 0) {
		return null
	}

	const itemsToBeDeleted = await Promise.allSettled(
		items.map((item) => ctx.db.get(item.id)),
	).then((results) =>
		results.filter(
			(result): result is PromiseFulfilledResult<Doc<"files">> =>
				result.status === "fulfilled" && result.value !== null,
		),
	)

	const deleteFilePromises = itemsToBeDeleted.map((item) =>
		Promise.all([
			ctx.db.delete(item.value._id),
			ctx.storage.delete(item.value.storageId),
		]),
	)

	const deleteResults = await Promise.allSettled(deleteFilePromises)

	const errors: Err.ApplicationErrorData[] = []
	let successfulDeletions = 0
	for (const result of deleteResults) {
		if (result.status === "rejected") {
			errors.push(Err.createJson(Err.Code.Internal))
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
		items: FileHandle[]
	},
) {
	if (items.length === 0) {
		return null
	}

	const itemsToBeRestored = await Promise.allSettled(
		items.map((item) => ctx.db.get(item.id)),
	).then((results) =>
		results.filter(
			(result): result is PromiseFulfilledResult<Doc<"files">> =>
				result.status === "fulfilled" && result.value !== null,
		),
	)

	const restoreFilePromises = itemsToBeRestored.map((item) =>
		ctx.db.patch(item.value._id, {
			deletedAt: undefined,
			updatedAt: Date.now(),
		}),
	)

	const restoreResults = await Promise.allSettled(restoreFilePromises)

	const errors: Err.ApplicationErrorData[] = []
	let successfulRestorations = 0
	for (const result of restoreResults) {
		if (result.status === "rejected") {
			errors.push(Err.createJson(Err.Code.Internal))
		} else {
			successfulRestorations += 1
		}
	}

	return { restored: successfulRestorations, errors }
}
