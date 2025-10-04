import { api } from "@fileone/convex/_generated/api"
import { createFileRoute } from "@tanstack/react-router"
import { useQuery as useConvexQuery } from "convex/react"
import { TrashIcon } from "lucide-react"
import { Button } from "@/components/ui/button"
import { DirectoryPageContext } from "@/directories/directory-page/context"
import { DirectoryPageSkeleton } from "@/directories/directory-page/directory-page-skeleton"
import { FilePathBreadcrumb } from "@/directories/directory-page/file-path-breadcrumb"

export const Route = createFileRoute(
	"/_authenticated/_sidebar-layout/trash/directories/$directoryId",
)({
	component: RouteComponent,
})

function RouteComponent() {
	const { directoryId } = Route.useParams()
	const rootDirectory = useConvexQuery(api.files.fetchRootDirectory)
	const directory = useConvexQuery(api.files.fetchDirectory, {
		directoryId,
	})
	const directoryContent = useConvexQuery(
		api.filesystem.fetchDirectoryContent,
		{
			directoryId,
			trashed: true,
		},
	)

	if (!directory || !directoryContent || !rootDirectory) {
		return <DirectoryPageSkeleton />
	}

	return (
		<DirectoryPageContext
			value={{ rootDirectory, directory, directoryContent }}
		>
			<header className="flex py-2 shrink-0 items-center gap-2 border-b px-4 w-full">
				<FilePathBreadcrumb />
				<div className="ml-auto flex flex-row gap-2">
					<EmptyTrashButton />
				</div>
			</header>

			{/* <DirectoryPage /> */}
		</DirectoryPageContext>
	)
}

function EmptyTrashButton() {
	return (
		<Button size="sm" type="button" variant="outline">
			<TrashIcon className="size-4" />
			Empty Trash
		</Button>
	)
}
