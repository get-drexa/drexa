import { queryOptions, skipToken } from "@tanstack/react-query"
import { fetchApi } from "@/lib/api"
import { Organization } from "./organization"
import { Drive } from "@/drive/drive"

export const organizationQuery = (orgSlug: string | null) =>
	queryOptions({
		queryKey: ["organizations", orgSlug],
		queryFn: orgSlug
			? () =>
					fetchApi("GET", `/organizations/${orgSlug}`, {
						returns: Organization,
					}).then(([, data]) => data)
			: skipToken,
	})

export const listOrganizationsQuery = queryOptions({
	queryKey: ["organizations"],
	queryFn: () =>
		fetchApi("GET", "/users/me/organizations", {
			returns: Organization.array(),
		}).then(([, data]) => data),
})

export const listOrganizationDrivesQuery = (orgSlug: string) =>
	queryOptions({
		queryKey: ["organizations", orgSlug, "drives"],
		queryFn: () =>
			fetchApi("GET", `/${orgSlug}/drives`, {
				returns: Drive.array(),
			}).then(([, data]) => data),
	})
