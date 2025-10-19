/**
 * An unhashed api key.
 * Always starts with sk, then the prefix specified at time of generation,
 * and ends with the base64url-encoded key.
 */
export type UnhashedApiKey = `sk-${ApiKeyPrefix}-${string}`

export type ApiKeyPrefix = string & { __brand: "ApiKeyPrefix" }

export type GenerateApiKeyOptions = {
	/**
	 * How long the key should be (excluding prefix) in bytes.
	 */
	keyByteLength: number

	/**
	 * Prefix of the api key. Can be a brief name of the consumer of the key.
	 * For example, if the prefix is "proxy", the key will be "sk-proxy-asdasjdjsdkjd.."
	 */
	prefix: ApiKeyPrefix

	expiresAt?: Date
	description: string
}

export type GenerateApiKeyResult = {
	unhashedKey: UnhashedApiKey
	hashedKey: string
	expiresAt?: Date
	description: string
}

export function newPrefix(prefix: string): ApiKeyPrefix | null {
	if (prefix.includes("-")) {
		return null
	}
	return prefix as ApiKeyPrefix
}

export async function generateApiKey({
	keyByteLength,
	prefix,
	expiresAt,
	description,
}: GenerateApiKeyOptions): Promise<GenerateApiKeyResult> {
	const keyContent = new Uint8Array(keyByteLength)
	crypto.getRandomValues(keyContent)

	const base64KeyContent = Buffer.from(keyContent).toString("base64url")
	const unhashedKey: UnhashedApiKey = `sk-${prefix}-${base64KeyContent}`

	const hashedKey = await Bun.password.hash(unhashedKey, {
		algorithm: "argon2id",
		memoryCost: 4, // memory usage in kibibytes
		timeCost: 3, // the number of iterations
	})

	return {
		unhashedKey,
		hashedKey,
		expiresAt,
		description,
	}
}
