import type { Id } from "@fileone/convex/dataModel"
import type {
	DirectoryHandle,
	DirectoryPathComponent,
} from "@fileone/convex/filesystem"
import type { DirectoryInfo } from "@fileone/convex/types"
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
	directory: DirectoryInfo
	rootLabel: string
	directoryUrlFn: (directory: Id<"directories">) => string
	fileDragInfoAtom?: PrimitiveAtom<FileDragInfo | null>
}) {
	const breadcrumbItems: React.ReactNode[] = [
		<FilePathBreadcrumbItem
			key={directory.path[0].handle.id}
			component={directory.path[0]}
			rootLabel={rootLabel}
			directoryUrlFn={directoryUrlFn}
			fileDragInfoAtom={fileDragInfoAtom}
		/>,
	]
	for (let i = 1; i < directory.path.length - 1; i++) {
		breadcrumbItems.push(
			<Fragment key={directory.path[i]?.handle.id}>
				<BreadcrumbSeparator />
				<FilePathBreadcrumbItem
					component={directory.path[i]!}
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
	component,
	rootLabel,
	directoryUrlFn,
	fileDragInfoAtom,
}: {
	component: DirectoryPathComponent
	rootLabel: string
	directoryUrlFn: (directory: Id<"directories">) => string
	fileDragInfoAtom: PrimitiveAtom<FileDragInfo | null>
}) {
	const { isDraggedOver, dropHandlers } = useFileDrop({
		destItem: component.handle as DirectoryHandle,
		dragInfoAtom: fileDragInfoAtom,
	})

	const dirName = component.name || rootLabel

	return (
		<Tooltip open={isDraggedOver}>
			<TooltipTrigger asChild>
				<BreadcrumbItem
					className={cn({ "bg-muted": isDraggedOver })}
					{...dropHandlers}
				>
					<BreadcrumbLink asChild>
						<Link to={directoryUrlFn(component.handle.id)}>
							{dirName}
						</Link>
					</BreadcrumbLink>
				</BreadcrumbItem>
			</TooltipTrigger>
			<TooltipContent>Move to {dirName}</TooltipContent>
		</Tooltip>
	)
}
