import { useQuery } from "@tanstack/react-query"
import { createFileRoute, Navigate } from "@tanstack/react-router"
import { LoadingSpinner } from "@/components/ui/loading-spinner"
import { listOrganizationsQuery } from "@/organization/api"
import { ORGANIZATION_KIND } from "@/organization/organization"

export const Route = createFileRoute("/_authenticated/")({
	component: RouteComponent,
	loader: ({ context }) => {
		void context.queryClient.ensureQueryData(listOrganizationsQuery)
	},
})

function RouteComponent() {
	const { data: orgs, isLoading } = useQuery(listOrganizationsQuery)

	if (isLoading) {
		return (
			<div className="flex h-screen w-full items-center justify-center">
				<LoadingSpinner className="size-10" />
			</div>
		)
	}

	const personalOrg = orgs?.find(
		(org) => org.kind === ORGANIZATION_KIND.personal,
	)

	if (!personalOrg) {
		return <Navigate replace to="/login" />
	}

	return (
		<Navigate
			replace
			to="/$orgSlug"
			params={{ orgSlug: personalOrg.slug }}
		/>
	)
}
