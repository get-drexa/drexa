import {
	verifyApiKey as _verifyApiKey,
	parseApiKey,
	WebCryptoSha256Hasher,
} from "@drexa/auth"
import type { MutationCtx, QueryCtx } from "../_generated/server"

export async function verifyApiKey(
	ctx: MutationCtx | QueryCtx,
	unhashedKey: string,
): Promise<boolean> {
	const parsedKey = parseApiKey(unhashedKey)
	if (!parsedKey) {
		return false
	}

	const apiKey = await ctx.db
		.query("apiKeys")
		.withIndex("byPublicId", (q) => q.eq("publicId", parsedKey.prefix))
		.first()
	if (!apiKey) {
		return false
	}

	return await _verifyApiKey({
		keyToBeVerified: parsedKey.unhashedKey,
		hashedKey: apiKey.hashedKey,
		hasher: new WebCryptoSha256Hasher(),
	})
}
