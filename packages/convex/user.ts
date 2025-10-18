import { authenticatedMutation } from "./functions"
import { ensureRootDirectory as ensureRootDirectoryImpl } from "./model/filesystem"

export const ensureRootDirectory = authenticatedMutation({
	handler: async (ctx) => {
		return await ensureRootDirectoryImpl(ctx)
	},
})
