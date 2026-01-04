import { createContext, useContext } from "react"
import type { Organization } from "./organization"

export const OrganizationContext = createContext<Organization>(
	null as unknown as Organization,
)

export function useCurrentOrganization() {
	const org = useContext(OrganizationContext)
	if (!org) {
		throw new Error(
			"useCurrentOrganization must be used under /$orgSlug routes",
		)
	}
	return org
}
