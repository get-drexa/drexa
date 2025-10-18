import { api } from "@fileone/convex/_generated/api"
import type { Doc, Id } from "@fileone/convex/_generated/dataModel"
import { useMutation as useConvexMutation } from "convex/react"
import { useCallback } from "react"

function useUploadFile({
	targetDirectory,
}: {
	targetDirectory: Doc<"directories">
}) {
	const generateUploadUrl = useConvexMutation(api.files.generateUploadUrl)
	const saveFile = useConvexMutation(api.files.saveFile)

	async function upload({
		file,
		onStart,
		onProgress,
	}: {
		file: File
		onStart: (xhr: XMLHttpRequest) => void
		onProgress: (progress: number) => void
	}) {
		const uploadUrl = await generateUploadUrl()

		return new Promise<{ storageId: Id<"_storage"> }>((resolve, reject) => {
			const xhr = new XMLHttpRequest()
			xhr.upload.addEventListener("progress", (e) => {
				onProgress(e.loaded / e.total)
			})
			xhr.upload.addEventListener("error", reject)
			xhr.addEventListener("load", () => {
				resolve(
					xhr.response as {
						storageId: Id<"_storage">
					},
				)
			})
			xhr.open("POST", uploadUrl)
			xhr.responseType = "json"
			xhr.setRequestHeader("Content-Type", file.type)
			xhr.send(file)
			onStart(xhr)
		}).then(({ storageId }) =>
			saveFile({
				storageId,
				name: file.name,
				size: file.size,
				mimeType: file.type,
				directoryId: targetDirectory._id,
			}),
		)
	}

	return useCallback(upload, [])
}

export default useUploadFile
