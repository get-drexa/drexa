import type { type } from "arktype"

export type RequestOptions = {
	signal?: AbortSignal
	headers?: HeadersInit
}

export type ClientOptions = {
	/** Absolute API base URL, including its path (for example https://host/api). */
	baseUrl: string | URL
	/** Set to include for the browser cookie flow. Defaults to omit. */
	credentials?: RequestCredentials
	/** Evaluated for each request so callers can rotate tokens. */
	accessToken?:
		| string
		| (() => string | undefined | Promise<string | undefined>)
	headers?: HeadersInit
	fetch?: (
		input: string | URL | Request,
		init?: RequestInit,
	) => Promise<Response>
}

export class ApiError extends Error {
	constructor(
		public readonly status: number,
		message: string,
		public readonly body?: unknown,
	) {
		super(message)
		this.name = "ApiError"
	}
}

export class Transport {
	private readonly baseUrl: URL
	private readonly fetcher: NonNullable<ClientOptions["fetch"]>

	constructor(private readonly options: ClientOptions) {
		this.baseUrl = new URL(options.baseUrl)
		if (!["http:", "https:"].includes(this.baseUrl.protocol)) {
			throw new TypeError("baseUrl must use HTTP or HTTPS")
		}
		this.baseUrl.search = ""
		this.baseUrl.hash = ""
		this.fetcher = options.fetch ?? globalThis.fetch.bind(globalThis)
	}

	url(
		path: string,
		query: Record<string, string | number | boolean | undefined> = {},
	): string {
		const url = new URL(this.baseUrl)
		url.pathname = `${url.pathname.replace(/\/+$/, "")}/${path.replace(/^\/+/, "")}`
		for (const [key, value] of Object.entries(query)) {
			if (value !== undefined) url.searchParams.set(key, String(value))
		}
		return url.toString()
	}

	async send(
		method: string,
		url: string,
		body: BodyInit | undefined,
		options: RequestOptions = {},
	): Promise<Response> {
		const headers = new Headers(this.options.headers)
		new Headers(options.headers).forEach((value, key) => {
			headers.set(key, value)
		})
		const token =
			typeof this.options.accessToken === "function"
				? await this.options.accessToken()
				: this.options.accessToken
		if (token) headers.set("Authorization", `Bearer ${token}`)
		const response = await this.fetcher(url, {
			method,
			body,
			headers,
			signal: options.signal,
			credentials: this.options.credentials ?? "omit",
		})
		if (!response.ok) {
			const text = await response.text()
			let errorBody: unknown = text
			try {
				errorBody = JSON.parse(text)
			} catch {
				/* Plain-text errors are also returned by Fiber. */
			}
			const message =
				typeof errorBody === "object" &&
				errorBody !== null &&
				"error" in errorBody &&
				typeof errorBody.error === "string"
					? errorBody.error
					: text || response.statusText
			throw new ApiError(response.status, message, errorBody)
		}
		return response
	}

	async json<T extends type.Any>(
		method: string,
		url: string,
		result: T,
		body?: unknown,
		options: RequestOptions = {},
	): Promise<T["infer"]> {
		const headers = new Headers(options.headers)
		headers.set("Accept", "application/json")
		if (body !== undefined) headers.set("Content-Type", "application/json")
		const response = await this.send(
			method,
			url,
			body === undefined ? undefined : JSON.stringify(body),
			{ ...options, headers },
		)
		return result.assert(await response.json())
	}

	async empty(
		method: string,
		url: string,
		options?: RequestOptions,
	): Promise<void> {
		await this.send(method, url, undefined, options)
	}
}

export function segment(value: string): string {
	// Dot segments are normalized by URL, even if percent encoded.
	if (!value || value === "." || value === "..")
		throw new TypeError("Invalid path identifier")
	return encodeURIComponent(value)
}
