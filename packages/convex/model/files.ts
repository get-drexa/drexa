import type { Id } from "../_generated/dataModel"
import type { AuthenticatedMutationCtx } from "../functions"
import * as Err from "./error"

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

	await ctx.db.patch(itemId, { name: newName })
}

export async function moveFiles(
	ctx: AuthenticatedMutationCtx,
	{
		targetDirectoryId,
		items,
	}: {
		targetDirectoryId: Id<"directories">
		items: Id<"files">[]
	},
) {
	const targetDirectory = await ctx.db.get(targetDirectoryId)
	if (!targetDirectory) {
		throw Err.create(
			Err.Code.DirectoryNotFound,
			"Target directory not found",
		)
	}
	await Promise.all(
		items.map((itemId) =>
			ctx.db.patch(itemId, { directoryId: targetDirectoryId }),
		),
	)
	return { items, targetDirectory }
}
