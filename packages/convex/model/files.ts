import type { Id } from "../_generated/dataModel"
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
			ctx.db.patch(handle.id, { directoryId: targetDirectoryHandle.id, updatedAt: Date.now() }),
		),
	)

	for (const updateResult of results) {
		if (updateResult.status === "rejected") {
			errors.push(Err.createJson(Err.Code.Internal))
		}
	}

	return { moved: okFiles, errors }
}
