import type { Doc } from "@fileone/convex/dataModel"
import { TextFileIcon } from "../components/icons/text-file-icon"
import { MiddleTruncatedText } from "../components/ui/middle-truncated-text"

export function FileGrid({ files }: { files: Doc<"files">[] }) {
	return (
		<div className="grid auto-cols-max grid-flow-col gap-4">
			{files.map((file) => (
				<div
					key={file._id}
					className="flex flex-col gap-2 items-center justify-center w-24"
				>
					<TextFileIcon className="size-10" />
					<MiddleTruncatedText>{file.name}</MiddleTruncatedText>
				</div>
			))}
		</div>
	)
}
