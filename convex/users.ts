import { mutation } from "./_generated/server"
import { authenticatedQuery } from "./functions"
import * as Err from "./model/error"

export const getCurrentUser = authenticatedQuery({
	handler: async (ctx) => {
		// ctx.user is the internal Convex user document
		return ctx.user
	},
})

export const syncUser = mutation({
	handler: async (ctx) => {
		const identity = await ctx.auth.getUserIdentity()
		if (!identity) {
			throw Err.create(Err.Code.Unauthenticated)
		}

		const existingUser = await ctx.db
			.query("users")
			.withIndex("byJwtSubject", (q) =>
				q.eq("jwtSubject", identity.subject),
			)
			.first()

		if (!existingUser) {
			await ctx.db.insert("users", {
				jwtSubject: identity.subject,
			})
		}
	},
})
