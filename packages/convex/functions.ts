import type {
	DocumentByName,
	TableNamesInDataModel,
	UserIdentity,
} from "convex/server"
import type { GenericId } from "convex/values"
import {
	customCtx,
	customMutation,
	customQuery,
} from "convex-helpers/server/customFunctions"
import type { DataModel } from "@fileone/convex/dataModel"
import type { MutationCtx, QueryCtx } from "@fileone/convex/server"
import { mutation, query } from "@fileone/convex/server"
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

/**
 * Gets a document by its id and checks if the user is authorized to access it
 *
 * @returns The document associated with the id or null if the document is not found.
 */
export async function authorizedGet<T extends TableNamesInDataModel<DataModel>>(
	ctx: AuthenticatedQueryCtx | AuthenticatedMutationCtx,
	id: GenericId<T>,
): Promise<DocumentByName<DataModel, T> | null> {
	const item = await ctx.db.get(id)
	if (item && item.userId !== ctx.user._id) {
		return null
	}
	return item
}
