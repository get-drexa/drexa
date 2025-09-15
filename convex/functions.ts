import {
	customMutation,
	customQuery,
} from "convex-helpers/server/customFunctions"
import { mutation, query } from "./_generated/server"
import { userIdentityOrThrow, userOrThrow } from "./model/user"

/**
 * Custom query that automatically provides authenticated user context
 * Throws an error if the user is not authenticated
 */
export const authenticatedQuery = customQuery(query, {
	args: {},
	input: async (ctx, args) => {
		const user = await userOrThrow(ctx)
		const identity = await userIdentityOrThrow(ctx)
		return {
			ctx: { ...ctx, user, identity },
			args,
		}
	},
})

/**
 * Custom mutation that automatically provides authenticated user context
 * Throws an error if the user is not authenticated
 */
export const authenticatedMutation = customMutation(mutation, {
	args: {},
	input: async (ctx, args) => {
		const user = await userOrThrow(ctx)
		const identity = await userIdentityOrThrow(ctx)

		return {
			ctx: { ...ctx, user, identity },
			args,
		}
	},
})
