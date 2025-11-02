import type { MutationCtx, QueryCtx } from "@fileone/convex/server"
import type { Doc } from "../_generated/dataModel"
import { authComponent } from "../auth"
import { type AuthenticatedQueryCtx, authorizedGet } from "../functions"
import * as Err from "../shared/error"

export type AuthUser = Awaited<ReturnType<typeof authComponent.getAuthUser>>

/**
 * Get the current authenticated user identity
 * Throws an error if the user is not authenticated */
export async function userIdentityOrThrow(ctx: QueryCtx | MutationCtx) {
	const identity = await ctx.auth.getUserIdentity()
	if (!identity) {
		throw Err.create(Err.Code.Unauthenticated, "Not authenticated")
	}
	return identity
}

/**
 * Get user document from JWT authentication
 * Throws an error if the user is not authenticated
 */
export async function userOrThrow(ctx: QueryCtx | MutationCtx) {
	const user = await authComponent.getAuthUser(ctx)
	return user
}

export async function queryInfo(ctx: AuthenticatedQueryCtx) {
	return await ctx.db
		.query("userInfo")
		.withIndex("byUserId", (q) => q.eq("userId", ctx.user._id))
		.first()
}
