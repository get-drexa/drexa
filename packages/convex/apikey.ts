import { verifyApiKey as _verifyApiKey, parseApiKey } from "@drexa/auth"
import { WebCryptoSha256Hasher } from "@drexa/auth/hasher"
import { v } from "convex/values"
import { query } from "./_generated/server"
import * as ApiKey from "./model/apikey"

export const verifyApiKey = query({
	args: {
		unhashedKey: v.string(),
	},
	handler: async (ctx, args) => {
		return await ApiKey.verifyApiKey(ctx, args.unhashedKey)
	},
})
