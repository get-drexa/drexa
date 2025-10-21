import { ConvexHttpClient } from "convex/browser"
import { createMiddleware } from "hono/factory"

const _client = new ConvexHttpClient(process.env.CONVEX_URL)

export type ConvexContextVariables = {
	convex: ConvexHttpClient
}

export const convexMiddleware = createMiddleware<{
	Variables: ConvexContextVariables
}>(async (c, next) => {
	c.var
	c.set("convex", _client)
	await next()
})
