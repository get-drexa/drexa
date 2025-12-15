import { Link } from "@tanstack/react-router"
import type { PrimitiveAtom } from "jotai"
import { atom } from "jotai"
import { Fragment } from "react"
import {
	Breadcrumb,
	BreadcrumbItem,
	BreadcrumbLink,
	BreadcrumbList,
	BreadcrumbPage,
	BreadcrumbSeparator,
} from "@/components/ui/breadcrumb"
import {
	Tooltip,
	TooltipContent,
	TooltipTrigger,
} from "@/components/ui/tooltip"
import type { FileDragInfo } from "@/files/use-file-drop"
import { useFileDrop } from "@/files/use-file-drop"
import { cn } from "@/lib/utils"
import type { DirectoryInfoWithPath } from "@/vfs/vfs"
import type { PathSegment } from "../lib/path"

/**
 * This is a placeholder file drag info atom that always stores null and is never mutated.
 */
const nullFileDragInfoAtom = atom<FileDragInfo | null>(null)

export function DirectoryPathBreadcrumb({
	directory,
	rootLabel,
	directoryUrlFn,
	fileDragInfoAtom = nullFileDragInfoAtom,
}: {
	directory: DirectoryInfoWithPath
	rootLabel: string
	directoryUrlFn: (directoryId: string) => string
	fileDragInfoAtom?: PrimitiveAtom<FileDragInfo | null>
}) {
	if (directory.path.length === 1) {
		return (
			<Breadcrumb>
				<BreadcrumbList>
					<BreadcrumbItem>
						<BreadcrumbPage>{rootLabel}</BreadcrumbPage>
					</BreadcrumbItem>
					<BreadcrumbSeparator />
				</BreadcrumbList>
			</Breadcrumb>
		)
	}

	const breadcrumbItems: React.ReactNode[] = [
		<FilePathBreadcrumbItem
			key={directory.path[0].id}
			segment={directory.path[0]}
			rootLabel={rootLabel}
			directoryUrlFn={directoryUrlFn}
			fileDragInfoAtom={fileDragInfoAtom}
		/>,
	]
	for (let i = 1; i < directory.path.length - 1; i++) {
		breadcrumbItems.push(
			<Fragment key={directory.path[i]!.id}>
				<BreadcrumbSeparator />
				<FilePathBreadcrumbItem
					segment={directory.path[i]!}
					rootLabel={rootLabel}
					directoryUrlFn={directoryUrlFn}
					fileDragInfoAtom={fileDragInfoAtom}
				/>
			</Fragment>,
		)
	}

	return (
		<Breadcrumb>
			<BreadcrumbList>
				{breadcrumbItems}
				<BreadcrumbSeparator />
				<BreadcrumbItem>
					<BreadcrumbPage>{directory.name}</BreadcrumbPage>{" "}
				</BreadcrumbItem>
			</BreadcrumbList>
		</Breadcrumb>
	)
}

function FilePathBreadcrumbItem({
	segment,
	rootLabel,
	directoryUrlFn,
	fileDragInfoAtom,
}: {
	segment: PathSegment
	rootLabel: string
	directoryUrlFn: (directoryId: string) => string
	fileDragInfoAtom: PrimitiveAtom<FileDragInfo | null>
}) {
	const { isDraggedOver, dropHandlers } = useFileDrop({
		destDir: segment.id,
		dragInfoAtom: fileDragInfoAtom,
	})

	const dirName = segment.name || rootLabel

	return (
		<Tooltip open={isDraggedOver}>
			<TooltipTrigger asChild>
				<BreadcrumbItem
					className={cn({ "bg-muted": isDraggedOver })}
					{...dropHandlers}
				>
					<BreadcrumbLink asChild>
						<Link to={directoryUrlFn(segment.id)}>{dirName}</Link>
					</BreadcrumbLink>
				</BreadcrumbItem>
			</TooltipTrigger>
			<TooltipContent>Move to {dirName}</TooltipContent>
		</Tooltip>
	)
}
