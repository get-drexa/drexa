import { listOrganizationDrivesQuery } from "@/organization/api"
import { useQuery } from "@tanstack/react-query"
import { createFileRoute } from "@tanstack/react-router"

export const Route = createFileRoute(
	"/_authenticated/$orgSlug/_sidebar-layout/drives/$driveId",
)({
	component: RouteComponent,
	loader: ({ context, params }) => {
		context.queryClient.ensureQueryData(
			listOrganizationDrivesQuery(params.orgSlug),
		)
	},
})

function RouteComponent() {
	const { orgSlug } = Route.useParams()
	const { data: drives } = useQuery(listOrganizationDrivesQuery(orgSlug))

	return (
		<div>
			Hello "/_authenticated/$orgSlug/_sidebar-layout/drives/$driveSlug"!
		</div>
	)
}
