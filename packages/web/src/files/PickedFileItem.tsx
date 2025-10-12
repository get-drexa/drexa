import { useAtomValue } from "jotai"
import { CircleAlertIcon, XIcon } from "lucide-react"
import type React from "react"
import { Button } from "@/components/ui/button"
import { Progress } from "@/components/ui/progress"
import { Tooltip } from "@/components/ui/tooltip"
import { FileUploadStatusKind, fileUploadStatusAtomFamily } from "./store"
import type { PickedFile } from "./upload-file-dialog"

export function PickedFileItem({
	file: pickedFile,
	onRemove,
}: {
	file: PickedFile
	onRemove: (file: PickedFile) => void
}) {
	const fileUploadAtom = fileUploadStatusAtomFamily(pickedFile.id)
	const fileUpload = useAtomValue(fileUploadAtom)
	console.log("fileUpload", fileUpload)
	const { file, id } = pickedFile

	let statusIndicator: React.ReactNode
	if (!fileUpload) {
		statusIndicator = (
			<Button
				variant="ghost"
				size="icon"
				onClick={() => onRemove(pickedFile)}
			>
				<XIcon className="size-4" />
			</Button>
		)
	} else {
		switch (fileUpload.kind) {
			case FileUploadStatusKind.InProgress:
				statusIndicator = <Progress value={fileUpload.progress * 100} />
				break
			case FileUploadStatusKind.Error:
				statusIndicator = (
					<Tooltip>
						<TooltipTrigger>
							<CircleAlertIcon />
						</TooltipTrigger>
					</Tooltip>
				)
		}
	}

	return (
		<li
			className="pl-3 pr-1 py-0.5 h-8 hover:bg-muted flex justify-between items-center"
			key={id}
		>
			<span>{file.name}</span>
			{fileUpload ? (
				<Progress
					className="max-w-20"
					value={fileUpload.progress * 100}
				/>
			) : (
				<Button
					variant="ghost"
					size="icon"
					onClick={() => onRemove(pickedFile)}
				>
					<XIcon className="size-4" />
				</Button>
			)}
		</li>
	)
}
