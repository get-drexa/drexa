import { v } from "convex/values"
import { apiKeyAuthenticatedQuery } from "./functions"
import * as FileShare from "./model/fileshare"

export const findFileShare = apiKeyAuthenticatedQuery({
	args: {
		shareToken: v.string(),
	},
	handler: async (ctx, { shareToken }) => {
		return await FileShare.find(ctx, { shareToken })
	},
})
