import type { FileInfo } from "./file"
import { ImagePreviewDialog } from "./image-preview-dialog"

export function FilePreviewDialog({
	openedFile,
	onClose,
}: {
	openedFile: FileInfo
	onClose: () => void
}) {
	switch (openedFile.mimeType) {
		case "image/jpeg":
		case "image/png":
		case "image/gif":
			return (
				<ImagePreviewDialog openedFile={openedFile} onClose={onClose} />
			)
		default:
			return null
	}
}
