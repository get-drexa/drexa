import type { Id } from "./_generated/dataModel"
import { query } from "./_generated/server"

export const getCurrentUser = query({
	handler: async (ctx) => {
		return await ctx.db.get(
			"jd7ampv4m200xr4yk2cfncccmh7qhj34" as Id<"users">,
		)
	},
})
