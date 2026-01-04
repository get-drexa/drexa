import { queryOptions, skipToken } from "@tanstack/react-query"
import { fetchApi } from "@/lib/api"
import { Drive } from "./drive"
import type { Organization } from "@/organization/organization"

export const drivesQuery = (org: Organization | null) =>
	queryOptions({
		queryKey: ["organizations", org?.slug, "drives"],
		queryFn: org
			? () =>
					fetchApi("GET", `/${org.slug}/drives`, {
						returns: Drive.array(),
					})
			: skipToken,
	})
