import { type } from "arktype"

export type ApiRoute =
	| "/auth/login"
	| "/auth/tokens"
	| "/accounts"
	| `/accounts/${string}`
	| `/accounts/${string}/uploads`
	| `/accounts/${string}/uploads/${string}/content`
	| `/accounts/${string}/uploads/${string}`
	| `/accounts/${string}/files${string}`
	| `/accounts/${string}/files/${string}`
	| `/accounts/${string}/files/${string}/content`
	| `/accounts/${string}/directories`
	| `/accounts/${string}/directories/${string}`
	| `/accounts/${string}/directories/${string}/content`
	| "/users/me"

export type HttpMethod = "GET" | "POST" | "PUT" | "DELETE" | "PATCH"

const baseApiUrl = new URL(
	import.meta.env.VITE_API_URL ??
		`${location.protocol}//${location.host}/api`,
)

export class ApiError extends Error {
	constructor(
		public readonly status: number,
		message: string,
	) {
		super(`api returned ${status}: ${message}`)
	}
}

export const Nothing = type({})
export type Nothing = typeof Nothing.infer

export async function fetchApi<Schema extends type.Any>(
	method: HttpMethod,
	route: ApiRoute,
	init: RequestInit & { returns: Schema },
): Promise<[response: Response, data: Schema["inferOut"]]> {
	let path: string
	if (baseApiUrl.pathname) {
		if (baseApiUrl.pathname.endsWith("/")) {
			path = `${baseApiUrl.pathname.slice(0, -1)}${route}`
		} else {
			path = `${baseApiUrl.pathname}${route}`
		}
	} else {
		path = route
	}
	const url = new URL(path, baseApiUrl)
	const response = await fetch(url, {
		credentials: "include",
		...init,
		method,
		headers: {
			"Content-Type": "application/json",
		},
	})
	if (!response.ok) {
		throw new ApiError(response.status, await response.text())
	}
	const body = await response.json()
	const result = init.returns(body)
	if (result instanceof type.errors) {
		throw result
	}
	return [response, result]
}
