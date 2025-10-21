import { api } from "@fileone/convex/api"
import { newRouter } from "./router"

const r = newRouter().basePath("/files")

r.get(":shareToken", async (c) => {
	const shareToken = c.req.param("shareToken")
	if (!shareToken) {
		return c.json({ error: "not found" }, 404)
	}

	const fileShare = await c.var.convex.query(api.fileshare.findFileShare, {
		apiKey: c.var.apiKey,
		shareToken,
	})
	if (!fileShare) {
		return c.json({ error: "not found" }, 404)
	}

	const fileUrl = await c.var.convex.query(api.filesystem.getStorageUrl, {
		apiKey: c.var.apiKey,
		storageId: fileShare.storageId,
	})
	if (!fileUrl) {
		return c.json({ error: "not found" }, 404)
	}

	const fileResponse = await fetch(fileUrl)
	if (!fileResponse.ok) {
		return c.json({ error: "not found" }, 404)
	}

	return new Response(fileResponse.body, {
		status: fileResponse.status,
		headers: fileResponse.headers,
	})
})

export { r as files }
