import type { RouterTypes } from "bun"

function router<
	R extends { [K in keyof R]: RouterTypes.RouteValue<Extract<K, string>> },
>(routes: R): R {
	return routes
}
