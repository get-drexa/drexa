import { mutationOptions, queryOptions, skipToken } from "@tanstack/react-query"
import { atom } from "jotai"
import { atomFamily } from "jotai/utils"
import { currentAccountAtom } from "@/account/account"
import { fetchApi, Nothing } from "@/lib/api"
import { Share } from "./share"

export const fileSharesQueryAtom = atomFamily((fileId: string) =>
	atom((get) => {
		const account = get(currentAccountAtom)
		return queryOptions({
			queryKey: ["accounts", account?.id, "shares", { fileId }],
			queryFn: account
				? () =>
						fetchApi(
							"GET",
							`/accounts/${account.id}/files/${fileId}/shares`,
							{ returns: Share.array() },
						).then(([_, result]) => result)
				: skipToken,
		})
	}),
)

export const directorySharesQueryAtom = atomFamily((directoryId: string) =>
	atom((get) => {
		const account = get(currentAccountAtom)
		return queryOptions({
			queryKey: ["accounts", account?.id, "shares", { directoryId }],
			queryFn: account
				? () =>
						fetchApi(
							"GET",
							`/accounts/${account.id}/directories/${directoryId}/shares`,
							{ returns: Share.array() },
						).then(([_, result]) => result)
				: skipToken,
		})
	}),
)

export const createShareMutationAtom = atom((get) =>
	mutationOptions({
		mutationFn: async ({ items }: { items: string[] }) => {
			const account = get(currentAccountAtom)
			if (!account) throw new Error("No account selected")

			const [_, result] = await fetchApi(
				"POST",
				`/accounts/${account.id}/shares`,
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
			const account = get(currentAccountAtom)
			if (!account) throw new Error("No account selected")

			await fetchApi(
				"DELETE",
				`/accounts/${account.id}/shares/${shareId}`,
				{ returns: Nothing },
			)
		},
	}),
)
