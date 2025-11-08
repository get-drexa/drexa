import { ConvexError } from "convex/values"
import type { Doc, Id } from "../_generated/dataModel"
import type { MutationCtx } from "../_generated/server"
import type {
	ApiKeyAuthenticatedQueryCtx,
	AuthenticatedMutationCtx,
	AuthenticatedQueryCtx,
} from "../functions"
import { ErrorCode, error } from "../shared/error"

export async function create(
	ctx: MutationCtx,
	{
		shareToken,
		storageId,
		expiresAt,
	}: { shareToken: string; storageId: Id<"_storage">; expiresAt?: Date },
) {
	const id = await ctx.db.insert("fileShares", {
		shareToken,
		storageId,
		expiresAt: expiresAt?.getTime(),
	})
	const doc = await ctx.db.get(id)
	if (!doc) {
		throw new ConvexError({ message: "Failed to create file share" })
	}
	return doc
}

export async function remove(
	ctx: AuthenticatedMutationCtx,
	{ doc }: { doc: Doc<"fileShares"> },
) {
	return await ctx.db.delete(doc._id)
}

export async function find(
	ctx:
		| AuthenticatedMutationCtx
		| AuthenticatedQueryCtx
		| ApiKeyAuthenticatedQueryCtx,
	{ shareToken }: { shareToken: string },
) {
	const doc = await ctx.db
		.query("fileShares")
		.withIndex("byShareToken", (q) => q.eq("shareToken", shareToken))
		.first()
	if (!doc) {
		error({
			code: ErrorCode.NotFound,
			message: "File share not found",
		})
	}

	if (hasExpired(doc)) {
		error({
			code: ErrorCode.NotFound,
			message: "File share not found",
		})
	}

	return doc
}

export async function findByStorageId(
	ctx: AuthenticatedMutationCtx | AuthenticatedQueryCtx,
	{ storageId }: { storageId: Id<"_storage"> },
) {
	return await ctx.db
		.query("fileShares")
		.withIndex("byStorageId", (q) => q.eq("storageId", storageId))
		.first()
}

export function hasExpired(doc: Doc<"fileShares">) {
	if (!doc.expiresAt) {
		return false
	}
	return doc.expiresAt < Date.now()
}

export async function updateExpiry(
	ctx: AuthenticatedMutationCtx,
	{ doc, expiresAt }: { doc: Doc<"fileShares">; expiresAt: Date },
) {
	return await ctx.db.patch(doc._id, {
		expiresAt: expiresAt.getTime(),
	})
}
