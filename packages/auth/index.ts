import { BunSha256Hasher, type PassswordHasher } from "./hasher"

export { WebCryptoSha256Hasher } from "./hasher"

/**
 * An unhashed api key.
 * Always starts with sk, then the prefix specified at time of generation,
 * and ends with the base64url-encoded key.
 */
export type UnhashedApiKey = `sk-${ApiKeyPrefix}-${string}`

export type ApiKeyPrefix = string & { __brand: "ApiKeyPrefix" }

export type ParsedApiKey = {
	prefix: ApiKeyPrefix
	unhashedKey: UnhashedApiKey
}

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

	hasher?: PassswordHasher
}

export type GenerateApiKeyResult = {
	unhashedKey: UnhashedApiKey
	hashedKey: string
	expiresAt?: Date
	description: string
}

export type VerifyApiKeyOptions = {
	keyToBeVerified: UnhashedApiKey
	hashedKey: string
	hasher?: PassswordHasher
}

function validatePrefix(prefix: string): prefix is ApiKeyPrefix {
	return !prefix.includes("-")
}

export function newPrefix(prefix: string): ApiKeyPrefix | null {
	if (!validatePrefix(prefix)) {
		return null
	}
	return prefix
}

export async function generateApiKey({
	keyByteLength,
	prefix,
	expiresAt,
	description,
	hasher = new BunSha256Hasher(),
}: GenerateApiKeyOptions): Promise<GenerateApiKeyResult> {
	const keyContent = new Uint8Array(keyByteLength)
	crypto.getRandomValues(keyContent)

	const base64KeyContent = Buffer.from(keyContent).toString("base64url")
	const unhashedKey: UnhashedApiKey = `sk-${prefix}-${base64KeyContent}`

	const hashedKey = await hasher.hash(unhashedKey)

	return {
		unhashedKey,
		hashedKey,
		expiresAt,
		description,
	}
}

export function parseApiKey(key: string): ParsedApiKey | null {
	if (!key.startsWith("sk-")) {
		return null
	}
	const parts = key.split("-")
	if (parts.length !== 3) {
		return null
	}
	const prefix = parts[1]
	if (!prefix || !validatePrefix(prefix)) {
		return null
	}
	return {
		prefix,
		unhashedKey: key as UnhashedApiKey,
	}
}

export async function verifyApiKey({
	keyToBeVerified,
	hashedKey,
	hasher = new BunSha256Hasher(),
}: VerifyApiKeyOptions): Promise<boolean> {
	return await hasher.verify(keyToBeVerified, hashedKey)
}
