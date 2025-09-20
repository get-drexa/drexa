import { api } from "@fileone/convex/_generated/api"
import type { Doc } from "@fileone/convex/_generated/dataModel"
import { DialogTitle } from "@radix-ui/react-dialog"
import { useQuery as useConvexQuery } from "convex/react"
import { atom, useAtom, useAtomValue, useSetAtom } from "jotai"
import {
	DownloadIcon,
	Maximize2Icon,
	Minimize2Icon,
	XIcon,
	ZoomInIcon,
	ZoomOutIcon,
} from "lucide-react"
import { useEffect, useRef } from "react"
import { Button } from "./ui/button"
import {
	Dialog,
	DialogClose,
	DialogContent,
	DialogHeader,
	DialogOverlay,
} from "./ui/dialog"
import { LoadingSpinner } from "./ui/loading-spinner"

const zoomLevelAtom = atom(
	1,
	(get, set, update: number | ((current: number) => number)) => {
		const current = get(zoomLevelAtom)
		console.log("current", current)
		const newValue = typeof update === "function" ? update(current) : update
		if (newValue >= 0.1) {
			set(zoomLevelAtom, newValue)
		}
	},
)

export function ImagePreviewDialog({
	file,
	onClose,
}: {
	file: Doc<"files">
	onClose: () => void
}) {
	const fileUrl = useConvexQuery(api.files.generateFileUrl, {
		storageId: file.storageId,
	})
	const setZoomLevel = useSetAtom(zoomLevelAtom)

	useEffect(
		() => () => {
			setZoomLevel(1)
		},
		[setZoomLevel],
	)

	return (
		<Dialog
			open
			onOpenChange={(open) => {
				if (!open) {
					onClose()
				}
			}}
		>
			<DialogOverlay className="flex items-center justify-center">
				{!fileUrl ? (
					<LoadingSpinner className="text-neutral-200 size-10" />
				) : null}
			</DialogOverlay>
			{fileUrl ? <PreviewContent fileUrl={fileUrl} file={file} /> : null}
		</Dialog>
	)
}

function PreviewContent({
	fileUrl,
	file,
}: {
	fileUrl: string
	file: Doc<"files">
}) {
	return (
		<DialogContent
			showCloseButton={false}
			className="p-0 lg:min-w-1/3 gap-0"
		>
			<DialogHeader className="overflow-auto border-b border-b-border p-4 flex flex-row items-center justify-between">
				<DialogTitle className="truncate flex-1">
					{file.name}
				</DialogTitle>
				<div className="flex flex-row items-center space-x-2">
					<Toolbar fileUrl={fileUrl} file={file} />
					<Button variant="ghost" size="icon" asChild>
						<DialogClose>
							<XIcon />
							<span className="sr-only">Close</span>
						</DialogClose>
					</Button>
				</div>
			</DialogHeader>
			<div className="w-full h-full flex items-center justify-center max-h-[calc(100vh-10rem)] overflow-auto">
				<ImagePreview fileUrl={fileUrl} file={file} />
			</div>
		</DialogContent>
	)
}

function Toolbar({ fileUrl, file }: { fileUrl: string; file: Doc<"files"> }) {
	const setZoomLevel = useSetAtom(zoomLevelAtom)
	const zoomInterval = useRef<ReturnType<typeof setInterval> | null>(null)

	useEffect(
		() => () => {
			if (zoomInterval.current) {
				clearInterval(zoomInterval.current)
				console.log("clearInterval")
				zoomInterval.current = null
			}
		},
		[],
	)

	function startZooming(delta: number) {
		setZoomLevel((zoom) => zoom + delta)
		zoomInterval.current = setInterval(() => {
			setZoomLevel((zoom) => zoom + delta)
		}, 100)
	}

	function stopZooming() {
		if (zoomInterval.current) {
			clearInterval(zoomInterval.current)
			zoomInterval.current = null
		}
	}

	return (
		<div className="flex flex-row items-center space-x-2 border-r border-r-border pr-4">
			<ResetZoomButton />
			<Button
				variant="ghost"
				onMouseDown={() => {
					startZooming(0.1)
				}}
				onMouseLeave={stopZooming}
				onMouseUp={stopZooming}
			>
				<ZoomInIcon />
			</Button>
			<Button
				variant="ghost"
				onMouseDown={() => {
					startZooming(-0.1)
				}}
				onMouseLeave={stopZooming}
				onMouseUp={stopZooming}
			>
				<ZoomOutIcon />
			</Button>
			<Button asChild>
				<a
					href={fileUrl}
					download={file.name}
					target="_blank"
					className="flex flex-row items-center"
				>
					<DownloadIcon />
					<span className="sr-only md:not-sr-only">Download</span>
				</a>
			</Button>
		</div>
	)
}

function ResetZoomButton() {
	const [zoomLevel, setZoomLevel] = useAtom(zoomLevelAtom)

	if (zoomLevel === 1) {
		return null
	}

	return (
		<Button
			variant="ghost"
			onClick={() => {
				setZoomLevel(1)
			}}
		>
			{zoomLevel > 1 ? <Minimize2Icon /> : <Maximize2Icon />}
		</Button>
	)
}

function ImagePreview({
	fileUrl,
	file,
}: {
	fileUrl: string
	file: Doc<"files">
}) {
	const zoomLevel = useAtomValue(zoomLevelAtom)
	return (
		<img
			src={fileUrl}
			alt={file.name}
			className="object-contain"
			style={{ transform: `scale(${zoomLevel})` }}
		/>
	)
}
