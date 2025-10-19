import { Hono } from "hono"

const h = new Hono().basePath("/files")

h.get("/:fileId", async (c) => {
	const fileId = c.req.param("fileId")
	if (!fileId) {
		return c.json({ error: "File ID is required" }, 400)
	}
})

export { h as files }
