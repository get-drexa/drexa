import type { Doc } from "@fileone/convex/_generated/dataModel"
import { ImagePreviewDialog } from "./image-preview-dialog"

export function FilePreviewDialog({
	file,
	onClose,
}: {
	file: Doc<"files">
	onClose: () => void
}) {
	if (!file) return null

	switch (file.mimeType) {
		case "image/jpeg":
		case "image/png":
		case "image/gif":
			return <ImagePreviewDialog file={file} onClose={onClose} />
		default:
			return null
	}
}
