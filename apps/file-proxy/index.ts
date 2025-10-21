import { Hono } from "hono"
import { apiKeyMiddleware } from "./auth"
import { convexMiddleware } from "./convex"
import { files } from "./files"

const app = new Hono()

app.use(convexMiddleware)
app.use(apiKeyMiddleware)

app.route("/", files)

export default {
	port: 8081,
	fetch: app.fetch,
}
