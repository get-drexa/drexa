import type { Doc, Id } from "../_generated/dataModel"
import type {
	AuthenticatedMutationCtx,
	AuthenticatedQueryCtx,
} from "../functions"
import * as FileShare from "./fileshare"

const PREVIEW_FILE_SHARE_VALID_FOR_MS = 1000 * 60 * 10 // 10 minutes

export async function find(
	ctx: AuthenticatedMutationCtx | AuthenticatedQueryCtx,
	{ storageId }: { storageId: Id<"_storage"> },
) {
	return await FileShare.findByStorageId(ctx, { storageId })
}

export async function create(
	ctx: AuthenticatedMutationCtx,
	{ storageId }: { storageId: Id<"_storage"> },
) {
	return await FileShare.create(ctx, {
		shareToken: crypto.randomUUID(),
		storageId,
		expiresAt: new Date(Date.now() + PREVIEW_FILE_SHARE_VALID_FOR_MS),
	})
}

export async function extend(
	ctx: AuthenticatedMutationCtx,
	{ doc }: { doc: Doc<"fileShares"> },
) {
	return await FileShare.updateExpiry(ctx, {
		doc,
		expiresAt: new Date(Date.now() + PREVIEW_FILE_SHARE_VALID_FOR_MS),
	})
}
