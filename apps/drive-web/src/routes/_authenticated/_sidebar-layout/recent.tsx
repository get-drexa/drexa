import { api } from "@fileone/convex/api"
import { createFileRoute } from "@tanstack/react-router"
import { useQuery as useConvexQuery } from "convex/react"
import { FileGrid } from "@/files/file-grid"

export const Route = createFileRoute("/_authenticated/_sidebar-layout/recent")({
	component: RouteComponent,
})

function RouteComponent() {
	const recentFiles = useConvexQuery(api.filesystem.fetchRecentFiles, {
		limit: 100,
	})

	console.log("recentFiles", recentFiles)

	return (
		<main className="p-4">
			<FileGrid files={recentFiles ?? []} />
		</main>
	)
}
