import { mutationOptions, queryOptions, skipToken } from "@tanstack/react-query"
import { fetchApi, Nothing } from "@/lib/api"
import { type Drive } from "@/drive/drive"
import type { Organization } from "@/organization/organization"
import { Share } from "./share"

export const fileSharesQuery = ({
	org,
	drive,
	fileId,
}: {
	org: Organization | null
	drive: Drive | null
	fileId: string
}) =>
	queryOptions({
		queryKey: [
			"organizations",
			org?.slug,
			"drives",
			drive?.id,
			"shares",
			{ fileId },
		],
		queryFn:
			org && drive
				? () =>
						fetchApi(
							"GET",
							`/${org.slug}/drives/${drive.id}/files/${fileId}/shares?includesExpired=true`,
							{ returns: Share.array() },
						).then(([_, result]) => result)
				: skipToken,
	})

export const directorySharesQuery = ({
	org,
	drive,
	directoryId,
}: {
	org: Organization | null
	drive: Drive | null
	directoryId: string
}) =>
	queryOptions({
		queryKey: [
			"organizations",
			org?.slug,
			"drives",
			drive?.id,
			"shares",
			{ directoryId },
		],
		queryFn:
			org && drive
				? () =>
						fetchApi(
							"GET",
							`/${org.slug}/drives/${drive.id}/directories/${directoryId}/shares?includesExpired=true`,
							{ returns: Share.array() },
						).then(([_, result]) => result)
					: skipToken,
	})

export const createShareMutationOptions = ({
	org,
	drive,
}: {
	org: Organization
	drive: Drive | null
}) =>
	mutationOptions({
		mutationFn: async ({ items }: { items: string[] }) => {
			if (!drive) throw new Error("No drive selected")

			const [_, result] = await fetchApi(
				"POST",
				`/${org.slug}/drives/${drive.id}/shares`,
				{
					body: JSON.stringify({ items }),
					returns: Share,
				},
			)

			return result
		},
	})

export const deleteShareMutationOptions = ({
	org,
	drive,
}: {
	org: Organization
	drive: Drive | null
}) =>
	mutationOptions({
		mutationFn: async ({ shareId }: { shareId: string }) => {
			if (!drive) throw new Error("No drive selected")

			await fetchApi(
				"DELETE",
				`/${org.slug}/drives/${drive.id}/shares/${shareId}`,
				{ returns: Nothing },
			)
		},
	})

export const updateShareMutationOptions = ({
	org,
	drive,
}: {
	org: Organization
	drive: Drive | null
}) =>
	mutationOptions({
		mutationFn: async ({
			shareId,
			expiresAt,
		}: {
			shareId: string
			expiresAt?: Date | null
		}) => {
			if (!drive) throw new Error("No drive selected")

			await fetchApi(
				"PATCH",
				`/${org.slug}/drives/${drive.id}/shares/${shareId}`,
				{ body: JSON.stringify({ expiresAt }), returns: Share },
			)
		},
	})
