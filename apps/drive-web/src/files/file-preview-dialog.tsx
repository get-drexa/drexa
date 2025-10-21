import type { OpenedFile } from "@fileone/convex/filesystem"
import { ImagePreviewDialog } from "./image-preview-dialog"

export function FilePreviewDialog({
	openedFile,
	onClose,
}: {
	openedFile: OpenedFile
	onClose: () => void
}) {
	switch (openedFile.file.mimeType) {
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
