import { Hono } from "hono"
import { handleFileRequest } from "./files"

Bun.serve({
	routes: {
		"/files/:fileId": {
			GET: handleFileRequest,
		},
	},
})
