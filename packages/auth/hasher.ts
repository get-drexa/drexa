export interface PassswordHasher {
	hash(password: string): Promise<string>
	verify(password: string, hash: string): Promise<boolean>
}

export class BunSha256Hasher implements PassswordHasher {
	async hash(password: string): Promise<string> {
		const hasher = new Bun.CryptoHasher("sha256")
		hasher.update(password)
		return hasher.digest("base64url")
	}

	async verify(password: string, hash: string): Promise<boolean> {
		const hasher = new Bun.CryptoHasher("sha256")
		hasher.update(password)

		const passwordHash = hasher.digest()
		const hashBytes = Buffer.from(hash, "base64url")

		if (passwordHash.byteLength !== hashBytes.byteLength) {
			return false
		}

		return crypto.timingSafeEqual(passwordHash, hashBytes)
	}
}

export class WebCryptoSha256Hasher implements PassswordHasher {
	async hash(password: string): Promise<string> {
		const hash = await crypto.subtle.digest(
			"SHA-256",
			new TextEncoder().encode(password),
		)
		return this.arrayBufferToBase64url(hash)
	}

	async verify(password: string, hash: string): Promise<boolean> {
		const passwordHash = await crypto.subtle.digest(
			"SHA-256",
			new TextEncoder().encode(password),
		)
		const hashBytes = this.base64urlToArrayBuffer(hash)

		if (passwordHash.byteLength !== hashBytes.byteLength) {
			return false
		}

		// Timing-safe comparison
		const passwordHashBytes = new Uint8Array(passwordHash)
		let result = 0
		for (let i = 0; i < passwordHashBytes.length; i++) {
			result |= passwordHashBytes[i]! ^ hashBytes[i]!
		}
		return result === 0
	}

	private arrayBufferToBase64url(buffer: ArrayBuffer): string {
		const bytes = new Uint8Array(buffer)
		let binary = ""
		for (let i = 0; i < bytes.length; i++) {
			binary += String.fromCharCode(bytes[i]!)
		}
		return btoa(binary).replace(/[+/=]/g, (char) => {
			switch (char) {
				case "+": return "-"
				case "/": return "_"
				case "=": return ""
				default: return char
			}
		})
	}

	private base64urlToArrayBuffer(base64url: string): Uint8Array {
		const base64 = base64url.replace(/-/g, "+").replace(/_/g, "/")

		const padded = base64.padEnd(
			base64.length + ((4 - (base64.length % 4)) % 4),
			"=",
		)

		const binary = atob(padded)
		const bytes = new Uint8Array(binary.length)
		for (let i = 0; i < binary.length; i++) {
			bytes[i] = binary.charCodeAt(i)
		}

		return bytes
	}
}
