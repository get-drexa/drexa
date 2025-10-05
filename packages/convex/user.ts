import { authenticatedMutation } from "./functions"
import * as FileSystem from "./model/filesystem"

export const ensureRootDirectory = authenticatedMutation({
	handler: async (ctx) => {
		return await FileSystem.ensureRootDirectory(ctx)
	},
})
