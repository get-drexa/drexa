import { api } from "@fileone/convex/_generated/api"
import { createFileRoute } from "@tanstack/react-router"
import { useQuery as useConvexQuery } from "convex/react"
import { DirectoryPageContext } from "@/directories/directory-page/context"
import { DirectoryPage } from "@/directories/directory-page/directory-page"
import { DirectoryPageSkeleton } from "@/directories/directory-page/directory-page-skeleton"

export const Route = createFileRoute(
	"/_authenticated/_sidebar-layout/directories/$directoryId",
)({
	component: RouteComponent,
})

function RouteComponent() {
	const { directoryId } = Route.useParams()
	const rootDirectory = useConvexQuery(api.files.fetchRootDirectory)
	const directory = useConvexQuery(api.files.fetchDirectory, {
		directoryId,
	})
	const directoryContent = useConvexQuery(api.files.fetchDirectoryContent, {
		directoryId,
	})

	if (!directory || !directoryContent || !rootDirectory) {
		return <DirectoryPageSkeleton />
	}

	return (
		<DirectoryPageContext
			value={{ rootDirectory, directory, directoryContent }}
		>
			<DirectoryPage />
		</DirectoryPageContext>
	)
}
