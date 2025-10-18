import type { MutationCtx, QueryCtx } from "@fileone/convex/server"
import { authComponent } from "../auth"
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
