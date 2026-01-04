import { useQuery, useQueryClient } from "@tanstack/react-query"
import { createFileRoute, Outlet } from "@tanstack/react-router"
import { SidebarInset, SidebarProvider } from "@/components/ui/sidebar"
import { LoadingSpinner } from "@/components/ui/loading-spinner"
import { DashboardSidebar } from "@/dashboard/dashboard-sidebar"
import { listOrganizationsQuery, organizationQuery } from "@/organization/api"
import { OrganizationContext } from "@/organization/context"

export const Route = createFileRoute(
	"/_authenticated/$orgSlug/_sidebar-layout",
)({
	component: RouteComponent,
	loader: ({ context, params }) => {
		context.queryClient.ensureQueryData(organizationQuery(params.orgSlug))
	},
})

function RouteComponent() {
	const { orgSlug } = Route.useParams()

	const client = useQueryClient()
	const { data: org, isLoading } = useQuery({
		...organizationQuery(orgSlug),
		initialData: () =>
			client
				.getQueryData(listOrganizationsQuery.queryKey)
				?.find((org) => org.slug === orgSlug),
	})

	if (isLoading) {
		return (
			<div className="flex h-screen w-full items-center justify-center">
				<LoadingSpinner className="size-10" />
			</div>
		)
	}

	if (!org) return null

	return (
		<OrganizationContext value={org}>
			<SidebarProvider>
				<div className="flex h-screen w-full">
					<DashboardSidebar />
					<SidebarInset className="overflow-hidden">
						<Outlet />
					</SidebarInset>
				</div>
			</SidebarProvider>
		</OrganizationContext>
	)
}
