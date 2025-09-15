import type { Id } from "../_generated/dataModel"
import type { MutationCtx, QueryCtx } from "../_generated/server"
import type { AuthenticatedMutationCtx } from "../functions"
import * as Err from "./error"

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
 * Get internal user document from JWT authentication
 * Throws an error if the user is not authenticated
 */
export async function userOrThrow(ctx: QueryCtx | MutationCtx) {
	const identity = await userIdentityOrThrow(ctx)

	// Look for existing user by JWT subject
	const user = await ctx.db
		.query("users")
		.withIndex("byJwtSubject", (q) => q.eq("jwtSubject", identity.subject))
		.first()

	if (!user) {
		throw Err.create(
			Err.Code.Unauthenticated,
			"User not found - please sync user first",
		)
	}

	return user
}

export async function register(ctx: AuthenticatedMutationCtx) {
	await ctx.db.insert("users", {
		jwtSubject: ctx.identity.subject,
	})
}
