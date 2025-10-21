import type { OpenedFile } from "@fileone/convex/filesystem"
import { DialogTitle } from "@radix-ui/react-dialog"
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
import { Button } from "@/components/ui/button"
import {
	Dialog,
	DialogClose,
	DialogContent,
	DialogHeader,
} from "@/components/ui/dialog"
import { fileShareUrl } from "./file-share"

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
	openedFile,
	onClose,
}: {
	openedFile: OpenedFile
	onClose: () => void
}) {
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
			<PreviewContent openedFile={openedFile} />
		</Dialog>
	)
}

function PreviewContent({ openedFile }: { openedFile: OpenedFile }) {
	return (
		<DialogContent
			showCloseButton={false}
			className="p-0 lg:min-w-1/3 gap-0"
		>
			<DialogHeader className="overflow-auto border-b border-b-border p-4 flex flex-row items-center justify-between">
				<DialogTitle className="truncate flex-1">
					{openedFile.file.name}
				</DialogTitle>
				<div className="flex flex-row items-center space-x-2">
					<Toolbar openedFile={openedFile} />
					<Button variant="ghost" size="icon" asChild>
						<DialogClose>
							<XIcon />
							<span className="sr-only">Close</span>
						</DialogClose>
					</Button>
				</div>
			</DialogHeader>
			<div className="w-full h-full flex items-center justify-center max-h-[calc(100vh-10rem)] overflow-auto">
				<ImagePreview openedFile={openedFile} />
			</div>
		</DialogContent>
	)
}

function Toolbar({ openedFile }: { openedFile: OpenedFile }) {
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
					href={fileShareUrl(openedFile.shareToken)}
					download={openedFile.file.name}
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

function ImagePreview({ openedFile }: { openedFile: OpenedFile }) {
	const zoomLevel = useAtomValue(zoomLevelAtom)
	return (
		<img
			src={fileShareUrl(openedFile.shareToken)}
			alt={openedFile.file.name}
			className="object-contain"
			style={{ transform: `scale(${zoomLevel})` }}
		/>
	)
}
