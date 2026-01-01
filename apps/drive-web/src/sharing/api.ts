import { mutationOptions, queryOptions, skipToken } from "@tanstack/react-query"
import { atom } from "jotai"
import { atomFamily } from "jotai/utils"
import { fetchApi, Nothing } from "@/lib/api"
import { currentDriveAtom } from "@/drive/drive"
import { Share } from "./share"

export const fileSharesQueryAtom = atomFamily((fileId: string) =>
	atom((get) => {
		const drive = get(currentDriveAtom)
		return queryOptions({
			queryKey: ["drives", drive?.id, "shares", { fileId }],
			queryFn: drive
				? () =>
						fetchApi(
							"GET",
							`/drives/${drive.id}/files/${fileId}/shares?includesExpired=true`,
							{ returns: Share.array() },
						).then(([_, result]) => result)
				: skipToken,
		})
	}),
)

export const directorySharesQueryAtom = atomFamily((directoryId: string) =>
	atom((get) => {
		const drive = get(currentDriveAtom)
		return queryOptions({
			queryKey: ["drives", drive?.id, "shares", { directoryId }],
			queryFn: drive
				? () =>
						fetchApi(
							"GET",
							`/drives/${drive.id}/directories/${directoryId}/shares?includesExpired=true`,
							{ returns: Share.array() },
						).then(([_, result]) => result)
				: skipToken,
		})
	}),
)

export const createShareMutationAtom = atom((get) =>
	mutationOptions({
		mutationFn: async ({ items }: { items: string[] }) => {
			const drive = get(currentDriveAtom)
			if (!drive) throw new Error("No drive selected")

			const [_, result] = await fetchApi(
				"POST",
				`/drives/${drive.id}/shares`,
				{
					body: JSON.stringify({ items }),
					returns: Share,
				},
			)

			return result
		},
	}),
)

export const deleteShareMutationAtom = atom((get) =>
	mutationOptions({
		mutationFn: async ({ shareId }: { shareId: string }) => {
			const drive = get(currentDriveAtom)
			if (!drive) throw new Error("No drive selected")

			await fetchApi(
				"DELETE",
				`/drives/${drive.id}/shares/${shareId}`,
				{ returns: Nothing },
			)
		},
	}),
)

export const updateShareMutationAtom = atom((get) =>
	mutationOptions({
		mutationFn: async ({
			shareId,
			expiresAt,
		}: {
			shareId: string
			expiresAt?: Date | null
		}) => {
			const drive = get(currentDriveAtom)
			if (!drive) throw new Error("No drive selected")

			await fetchApi(
				"PATCH",
				`/drives/${drive.id}/shares/${shareId}`,
				{ body: JSON.stringify({ expiresAt }), returns: Share },
			)
		},
	}),
)
