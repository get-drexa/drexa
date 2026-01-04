import { type } from "arktype"

export const ORGANIZATION_KIND = {
	personal: "personal",
	team: "team",
} as const
export type OrganizationKind =
	(typeof ORGANIZATION_KIND)[keyof typeof ORGANIZATION_KIND]

export const Organization = type({
	id: "string",
	kind: type.valueOf(ORGANIZATION_KIND),
	name: "string",
	slug: "string",
	createdAt: "string.date.iso.parse",
	updatedAt: "string.date.iso.parse",
})
export type Organization = typeof Organization.infer
