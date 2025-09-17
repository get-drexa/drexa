import { joinPath, PATH_SEPARATOR } from "@fileone/path"
import { createFileRoute } from "@tanstack/react-router"
import { FilesPage } from "@/files/files-page"

export const Route = createFileRoute("/_authenticated/_sidebar-layout/files/$")(
	{
		component: RouteComponent,
	},
)

function RouteComponent() {
	const { _splat } = Route.useParams()
	const path = _splat ? joinPath("", _splat) : PATH_SEPARATOR
	return <FilesPage path={path} />
}
