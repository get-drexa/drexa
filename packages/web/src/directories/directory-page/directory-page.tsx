import { useAtom } from "jotai"
import { ImagePreviewDialog } from "@/components/image-preview-dialog"
import { DirectoryContentTable } from "./directory-content-table"
import { openedFileAtom } from "./state"

export function DirectoryPage() {
	return (
		<>
			<div className="w-full">
				<DirectoryContentTable />
			</div>
			<PreviewDialog />
		</>
	)
}

function PreviewDialog() {
	const [openedFile, setOpenedFile] = useAtom(openedFileAtom)

	if (!openedFile) return null

	switch (openedFile.mimeType) {
		case "image/jpeg":
		case "image/png":
		case "image/gif":
			return (
				<ImagePreviewDialog
					file={openedFile}
					onClose={() => setOpenedFile(null)}
				/>
			)
		default:
			return null
	}
}
