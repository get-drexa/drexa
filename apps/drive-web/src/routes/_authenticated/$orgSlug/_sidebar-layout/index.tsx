import { createFileRoute } from "@tanstack/react-router"
import { useCurrentOrganization } from "@/organization/context"

export const Route = createFileRoute(
	"/_authenticated/$orgSlug/_sidebar-layout/",
)({
	component: RouteComponent,
})

function RouteComponent() {
	const org = useCurrentOrganization()
	return <div>Home of organization "{org.name}"</div>
}
