import type { DataModel } from "@fileone/convex/dataModel"
import type { MutationCtx, QueryCtx } from "@fileone/convex/server"
import { mutation, query } from "@fileone/convex/server"
import type {
	DocumentByName,
	TableNamesInDataModel,
	UserIdentity,
} from "convex/server"
import { type GenericId, v } from "convex/values"
import {
	customCtx,
	customMutation,
	customQuery,
} from "convex-helpers/server/customFunctions"
import * as ApiKey from "./model/apikey"
import { type AuthUser, userIdentityOrThrow, userOrThrow } from "./model/user"
import * as Err from "./shared/error"

export type AuthenticatedQueryCtx = QueryCtx & {
	user: AuthUser
	identity: UserIdentity
}

export type AuthenticatedMutationCtx = MutationCtx & {
	user: AuthUser
	identity: UserIdentity
}

export type ApiKeyAuthenticatedQueryCtx = QueryCtx & {
	__branded: "ApiKeyAuthenticatedQueryCtx"
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
 * Custom query that requires api key authentication for a query.
 */
export const apiKeyAuthenticatedQuery = customQuery(query, {
	args: {
		apiKey: v.string(),
	},
	input: async (ctx, args) => {
		if (!(await ApiKey.verifyApiKey(ctx, args.apiKey))) {
			throw Err.create(Err.Code.Unauthenticated, "Invalid API key")
		}
		return { ctx: ctx as ApiKeyAuthenticatedQueryCtx, args }
	},
})

/**
 * Custom mutation that requires api key authentication for a mutation.
 */
export const apiKeyAuthenticatedMutation = customMutation(mutation, {
	args: {
		apiKey: v.string(),
	},
	input: async (ctx, args) => {
		if (!(await ApiKey.verifyApiKey(ctx, args.apiKey))) {
			throw Err.create(Err.Code.Unauthenticated, "Invalid API key")
		}
		return { ctx, args }
	},
})

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
