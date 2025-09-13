import type { Id } from "./_generated/dataModel"
import { query } from "./_generated/server"

export const getCurrentUser = query({
	handler: async (ctx) => {
		return await ctx.db.get(
			"j574n657f521n19v1stnr88ysd7qhbs1" as Id<"users">,
		)
	},
})
