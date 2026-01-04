import { type } from "arktype"

export type ApiRoute =
	| "/auth/login"
	| "/auth/tokens"
	| "/accounts"
	| `/accounts/${string}`
	| `/${string}/drives`
	| `/${string}/drives/${string}`
	| `/${string}/drives/${string}/uploads`
	| `/${string}/drives/${string}/uploads/${string}/content`
	| `/${string}/drives/${string}/uploads/${string}`
	| `/${string}/drives/${string}/files${string}`
	| `/${string}/drives/${string}/files/${string}`
	| `/${string}/drives/${string}/files/${string}/content`
	| `/${string}/drives/${string}/files/${string}/shares${string}`
	| `/${string}/drives/${string}/directories${string}`
	| `/${string}/drives/${string}/directories/${string}`
	| `/${string}/drives/${string}/directories/${string}/content`
	| `/${string}/drives/${string}/directories/${string}/shares${string}`
	| `/${string}/drives/${string}/shares`
	| `/${string}/drives/${string}/shares/${string}`
	| `/shares/${string}`
	| `/shares/${string}/directories${string}`
	| `/shares/${string}/files${string}`
	| `/organizations/${string}`
	| "/users/me"
	| "/users/me/organizations"

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

export function buildApiUrl(route: ApiRoute): URL {
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
	return new URL(path, baseApiUrl)
}

export async function fetchApi<Schema extends type.Any>(
	method: HttpMethod,
	route: ApiRoute,
	init: RequestInit & { returns: Schema },
): Promise<[response: Response, data: Schema["inferOut"]]> {
	const url = buildApiUrl(route)
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
	// @ts-expect-error if the response returns nothing, then init.returns *is Nothing*, but typescript thinks its not right for some reason
	if (init.returns !== Nothing) {
		const body = await response.json()
		const result = init.returns(body)
		if (result instanceof type.errors) {
			throw result
		}
		return [response, result]
	}
	return [response, null]
}
