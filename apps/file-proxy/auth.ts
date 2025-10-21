import { createMiddleware } from "hono/factory"

export type ApiKeyContextVariable = {
	apiKey: string
}

const apiKeyMiddleware = createMiddleware<{ Variables: ApiKeyContextVariable }>(
	async (c, next) => {
		c.set("apiKey", process.env.API_KEY)
		await next()
	},
)

export { apiKeyMiddleware }
