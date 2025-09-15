import { mutation } from "./_generated/server"
import { authenticatedQuery } from "./functions"
import { getOrCreateUser, userIdentityOrThrow } from "./model/user"

export const getCurrentUser = authenticatedQuery({
	handler: async (ctx) => {
		// ctx.user is the internal Convex user document
		return ctx.user
	},
})

export const syncUser = mutation({
	handler: async (ctx) => {
		// This function creates or updates the internal user from identity provider
		const userId = await getOrCreateUser(ctx)
		const identity = await userIdentityOrThrow(ctx)
		
		return {
			userId,
			jwtSubject: identity.subject,
			name: identity.name,
			email: identity.email,
		}
	},
})
