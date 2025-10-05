import type { UserIdentity } from "convex/server"
import {
	customCtx,
	customMutation,
	customQuery,
} from "convex-helpers/server/customFunctions"
import type { MutationCtx, QueryCtx } from "./_generated/server"
import { mutation, query } from "./_generated/server"
import { type AuthUser, userIdentityOrThrow, userOrThrow } from "./model/user"

export type AuthenticatedQueryCtx = QueryCtx & {
	user: AuthUser
	identity: UserIdentity
}

export type AuthenticatedMutationCtx = MutationCtx & {
	user: AuthUser
	identity: UserIdentity
}

/**
 * Custom query that automatically provides authenticated user context
 * Throws an error if the user is not authenticated
 */
export const authenticatedQuery = customQuery(
	query,
	customCtx(async (ctx: QueryCtx) => {
		const user = await userOrThrow(ctx)
		const identity = await userIdentityOrThrow(ctx)
		return { user, identity }
	}),
)

/**
 * Custom mutation that automatically provides authenticated user context
 * Throws an error if the user is not authenticated
 */
export const authenticatedMutation = customMutation(
	mutation,
	customCtx(async (ctx: MutationCtx) => {
		const user = await userOrThrow(ctx)
		const identity = await userIdentityOrThrow(ctx)
		return { user, identity }
	}),
)
