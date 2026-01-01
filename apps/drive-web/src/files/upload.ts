import { type } from "arktype"
import { ApiError, fetchApi } from "@/lib/api"
import type { Drive } from "@/drive/drive"
import type { DirectoryInfoWithPath } from "@/vfs/vfs"

export const UploadStatus = type.enumerated("pending", "completed", "failed")
export type UploadStatus = typeof UploadStatus.infer

export const Upload = type({
	id: "string",
	status: UploadStatus,
	uploadUrl: "string.url",
})
export type Upload = typeof Upload.infer

export async function uploadFile({
	drive,
	file,
	targetDirectory,
	onStart,
	onProgress,
}: {
	drive: Drive
	file: File
	targetDirectory: DirectoryInfoWithPath
	onStart: (xhr: XMLHttpRequest) => void
	onProgress: (progress: number) => void
}) {
	const [, upload] = await fetchApi(
		"POST",
		`/drives/${drive.id}/uploads`,
		{
			body: JSON.stringify({
				name: file.name,
				parentId: targetDirectory.id,
			}),
			returns: Upload,
		},
	)

	await putFile({
		file,
		uploadUrl: upload.uploadUrl,
		onStart,
		onProgress,
	})

	await fetchApi("PATCH", `/drives/${drive.id}/uploads/${upload.id}`, {
		body: JSON.stringify({
			status: "completed",
		}),
		returns: Upload,
	})

	return upload
}

function putFile({
	file,
	uploadUrl,
	onStart,
	onProgress,
}: {
	file: File
	uploadUrl: string
	onStart: (xhr: XMLHttpRequest) => void
	onProgress: (progress: number) => void
}): Promise<void> {
	return new Promise<void>((resolve, reject) => {
		const xhr = new XMLHttpRequest()
		xhr.withCredentials = true
		xhr.upload.addEventListener("progress", (e) => {
			onProgress(e.loaded / e.total)
		})
		xhr.upload.addEventListener("error", reject)
		xhr.addEventListener("load", () => {
			if (xhr.status === 200 || xhr.status === 204) {
				resolve()
			} else {
				reject(new ApiError(xhr.status, xhr.response))
			}
		})
		xhr.open("PUT", uploadUrl)
		xhr.responseType = "json"
		xhr.setRequestHeader("Content-Type", file.type)
		xhr.send(file)
		onStart(xhr)
	})
}
