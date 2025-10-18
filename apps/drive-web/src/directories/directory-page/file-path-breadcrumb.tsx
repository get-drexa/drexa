import type { Id } from "@fileone/convex/dataModel"
import type {
	DirectoryHandle,
	DirectoryPathComponent,
} from "@fileone/convex/filesystem"
import { Link } from "@tanstack/react-router"
import { Fragment, useContext } from "react"
import {
	Breadcrumb,
	BreadcrumbItem,
	BreadcrumbLink,
	BreadcrumbList,
	BreadcrumbPage,
	BreadcrumbSeparator,
} from "../../components/ui/breadcrumb"
import {
	Tooltip,
	TooltipContent,
	TooltipTrigger,
} from "../../components/ui/tooltip"
import { useFileDrop } from "../../files/use-file-drop"
import { cn } from "../../lib/utils"
import { DirectoryPageContext } from "./context"
import { dragInfoAtom } from "./state"

export function FilePathBreadcrumb({
	rootLabel,
	directoryUrlFn,
}: {
	rootLabel: string
	directoryUrlFn: (directory: Id<"directories">) => string
}) {
	const { rootDirectory, directory } = useContext(DirectoryPageContext)

	const breadcrumbItems: React.ReactNode[] = []
	for (let i = 1; i < directory.path.length - 1; i++) {
		breadcrumbItems.push(
			<Fragment key={directory.path[i]?.handle.id}>
				<BreadcrumbSeparator />
				<FilePathBreadcrumbItem
					component={directory.path[i]!}
					rootLabel={rootLabel}
					directoryUrlFn={directoryUrlFn}
				/>
			</Fragment>,
		)
	}

	return (
		<Breadcrumb>
			<BreadcrumbList>
				{rootDirectory._id === directory._id ? (
					<BreadcrumbItem>
						<BreadcrumbPage>{rootLabel}</BreadcrumbPage>
					</BreadcrumbItem>
				) : (
					<FilePathBreadcrumbItem
						component={directory.path[0]!}
						rootLabel={rootLabel}
						directoryUrlFn={directoryUrlFn}
					/>
				)}
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
}: {
	component: DirectoryPathComponent
	rootLabel: string
	directoryUrlFn: (directory: Id<"directories">) => string
}) {
	const { isDraggedOver, dropHandlers } = useFileDrop({
		destItem: component.handle as DirectoryHandle,
		dragInfoAtom,
	})

	const dirName = component.name || rootLabel

	console.log({ dirName, isDraggedOver, dropHandlers })

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
