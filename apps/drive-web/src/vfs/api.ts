import {
	type InfiniteData,
	infiniteQueryOptions,
	mutationOptions,
	queryOptions,
	skipToken,
} from "@tanstack/react-query"
import { type } from "arktype"
import { atom } from "jotai"
import { atomFamily } from "jotai/utils"
import { currentAccountAtom } from "@/account/account"
import { fetchApi } from "@/lib/api"
import {
	DirectoryContent,
	DirectoryInfo,
	DirectoryInfoWithPath,
	DirectoryItem,
	FileInfo,
} from "./vfs"

const DirectoryContentResponse = type({
	items: DirectoryContent,
	"nextCursor?": "string",
})

/**
 * This atom derives the file url for a given file.
 * It is recommended to use {@link useFileUrl} instead of using this atom directly.
 */
export const fileUrlAtom = atomFamily((fileId: string) =>
	atom((get) => {
		const account = get(currentAccountAtom)
		if (!account) {
			return ""
		}
		return `${import.meta.env.VITE_API_URL}/accounts/${account.id}/files/${fileId}/content`
	}),
)

export const rootDirectoryQueryAtom = atom((get) => {
	const account = get(currentAccountAtom)
	return queryOptions({
		queryKey: ["accounts", account?.id, "directories", "root"],
		queryFn: account
			? () =>
					fetchApi(
						"GET",
						`/accounts/${account.id}/directories/root?include=path`,
						{ returns: DirectoryInfoWithPath },
					).then(([_, result]) => result)
			: skipToken,
	})
})

export const directoryInfoQueryAtom = atomFamily((directoryId: string) =>
	atom((get) => {
		const account = get(currentAccountAtom)
		return queryOptions({
			queryKey: ["accounts", account?.id, "directories", directoryId],
			queryFn: account
				? () =>
						fetchApi(
							"GET",
							`/accounts/${account.id}/directories/${directoryId}?include=path`,
							{ returns: DirectoryInfoWithPath },
						).then(([_, result]) => result)
				: skipToken,
		})
	}),
)

export const DIRECTORY_CONTENT_ORDER_BY = {
	name: "name",
	createdAt: "createdAt",
	updatedAt: "updatedAt",
} as const
export type DirectoryContentOrderBy =
	(typeof DIRECTORY_CONTENT_ORDER_BY)[keyof typeof DIRECTORY_CONTENT_ORDER_BY]

export const DIRECTORY_CONTENT_ORDER_DIRECTION = {
	asc: "asc",
	desc: "desc",
} as const
type DirectoryContentOrderDirection =
	(typeof DIRECTORY_CONTENT_ORDER_DIRECTION)[keyof typeof DIRECTORY_CONTENT_ORDER_DIRECTION]

type DirectoryContentQueryParams = {
	directoryId: string
	orderBy: DirectoryContentOrderBy
	direction: DirectoryContentOrderDirection
	limit: number
}

const directoryContentQueryKey = (
	accountId: string | undefined,
	directoryId: string,
) => ["accounts", accountId, "directories", directoryId, "content"]
export const directoryContentQueryAtom = atomFamily(
	({ directoryId, orderBy, direction, limit }: DirectoryContentQueryParams) =>
		atom((get) => {
			const account = get(currentAccountAtom)
			return infiniteQueryOptions({
				queryKey: directoryContentQueryKey(account?.id, directoryId),
				initialPageParam: {
					orderBy,
					direction,
					limit,
					cursor: "",
				},
				queryFn: ({ pageParam }) =>
					account
						? fetchApi(
								"GET",
								`/accounts/${account.id}/directories/${directoryId}/content?orderBy=${pageParam.orderBy}&dir=${pageParam.direction}&limit=${pageParam.limit}${pageParam.cursor ? `&cursor=${pageParam.cursor}` : ""}`,
								{ returns: DirectoryContentResponse },
							).then(([_, result]) => result)
						: Promise.reject(new Error("No account selected")),
				getNextPageParam: (lastPage, _pages, lastPageParam) => ({
					...lastPageParam,
					cursor: lastPage.nextCursor ?? "",
				}),
			})
		}),
	(paramsA, paramsB) =>
		paramsA.directoryId === paramsB.directoryId &&
		paramsA.orderBy === paramsB.orderBy &&
		paramsA.direction === paramsB.direction &&
		paramsA.limit === paramsB.limit,
)

export const createDirectoryMutationAtom = atom((get) => {
	const account = get(currentAccountAtom)
	return mutationOptions({
		mutationFn: async (data: { name: string; parentId: string }) => {
			if (!account) throw new Error("No account selected")
			return fetchApi(
				"POST",
				`/accounts/${account.id}/directories?include=path`,
				{
					body: JSON.stringify({
						name: data.name,
						parentId: data.parentId,
					}),
					returns: DirectoryInfoWithPath,
				},
			).then(([_, result]) => result)
		},
		onSuccess: (data, _variables, _context, { client }) => {
			client.setQueryData(
				get(directoryInfoQueryAtom(data.id)).queryKey,
				data,
			)
		},
	})
})

export const MoveDirectoryItemsResult = type({
	items: DirectoryItem.array(),
	moved: "string[]",
	conflicts: "string[]",
	errors: type({
		id: "string",
		error: "string",
	}).array(),
})
export type MoveDirectoryItemsResult = typeof MoveDirectoryItemsResult.infer

export const moveDirectoryItemsMutationAtom = atom((get) =>
	mutationOptions({
		mutationFn: async ({
			targetDirectory,
			items,
		}: {
			targetDirectory: DirectoryInfo | string
			items: DirectoryItem[]
		}) => {
			const account = get(currentAccountAtom)
			if (!account) {
				throw new Error("Account not found")
			}

			const dirId =
				typeof targetDirectory === "string"
					? targetDirectory
					: targetDirectory.id

			const [, result] = await fetchApi(
				"POST",
				`/accounts/${account.id}/directories/${dirId}/content`,
				{
					body: JSON.stringify({
						items: items.map((item) => item.id),
					}),
					returns: MoveDirectoryItemsResult,
				},
			)
			return result
		},
		onMutate: ({ items }, { client }) => {
			const account = get(currentAccountAtom)
			if (!account) {
				return null
			}

			const movedItems = new Map<string, Set<string>>()

			for (const item of items) {
				if (item.parentId) {
					const s = movedItems.get(item.parentId)
					if (!s) {
						movedItems.set(item.parentId, new Set([item.id]))
					} else {
						s.add(item.id)
					}
				}
			}

			const prevDirContentMap = new Map<
				string,
				InfiniteData<typeof DirectoryContentResponse.infer> | undefined
			>()

			movedItems.forEach((s, parentId) => {
				const key = directoryContentQueryKey(account.id, parentId)
				const prevDirContent =
					client.getQueryData<
						InfiniteData<typeof DirectoryContentResponse.infer>
					>(key)
				client.setQueryData<
					InfiniteData<typeof DirectoryContentResponse.infer>
				>(key, (prev) => {
					if (!prev) return prev
					return {
						...prev,
						pages: prev.pages.map((page) => ({
							...page,
							items: page.items.filter((it) => !s.has(it.id)),
						})),
					}
				})
				prevDirContentMap.set(parentId, prevDirContent)
			})

			return { prevDirContentMap }
		},
		onSuccess: (_data, { targetDirectory, items }, _result, { client }) => {
			const account = get(currentAccountAtom)
			if (!account) return

			const dirId =
				typeof targetDirectory === "string"
					? targetDirectory
					: targetDirectory.id
			client.invalidateQueries({
				queryKey: directoryContentQueryKey(account.id, dirId),
			})
			for (const item of items) {
				if (item.parentId) {
					client.invalidateQueries({
						queryKey: directoryContentQueryKey(
							account.id,
							item.parentId,
						),
					})
				}
			}
		},
		onError: (_error, _vars, context, { client }) => {
			if (context) {
				const account = get(currentAccountAtom)
				if (account) {
					context.prevDirContentMap.forEach(
						(prevDirContent, parentId) => {
							client.setQueryData(
								directoryContentQueryKey(account.id, parentId),
								prevDirContent,
							)
						},
					)
				}
			}
		},
	}),
)

export const moveToTrashMutationAtom = atom((get) =>
	mutationOptions({
		mutationFn: async (items: DirectoryItem[]) => {
			const account = get(currentAccountAtom)
			if (!account) {
				throw new Error("Account not found")
			}

			const fileIds: string[] = []
			const directoryIds: string[] = []
			for (const item of items) {
				switch (item.kind) {
					case "file":
						fileIds.push(item.id)
						break
					case "directory":
						directoryIds.push(item.id)
						break
				}
			}

			let deleteFilesPromise: Promise<FileInfo[]>
			if (fileIds.length > 0) {
				const fileDeleteParams = new URLSearchParams()
				fileDeleteParams.set("id", fileIds.join(","))
				fileDeleteParams.set("trash", "true")
				deleteFilesPromise = fetchApi(
					"DELETE",
					`/accounts/${account.id}/files?${fileDeleteParams.toString()}`,
					{
						returns: FileInfo.array(),
					},
				).then(([_, result]) => result)
			} else {
				deleteFilesPromise = Promise.resolve([])
			}

			let deleteDirectoriesPromise: Promise<DirectoryInfo[]>
			if (directoryIds.length > 0) {
				const directoryDeleteParams = new URLSearchParams()
				directoryDeleteParams.set("id", directoryIds.join(","))
				directoryDeleteParams.set("trash", "true")
				deleteDirectoriesPromise = fetchApi(
					"DELETE",
					`/accounts/${account.id}/directories?${directoryDeleteParams.toString()}`,
					{
						returns: DirectoryInfo.array(),
					},
				).then(([_, result]) => result)
			} else {
				deleteDirectoriesPromise = Promise.resolve([])
			}

			const [deletedFiles, deletedDirectories] = await Promise.all([
				deleteFilesPromise,
				deleteDirectoriesPromise,
			])

			return [...deletedFiles, ...deletedDirectories]
		},
		onMutate: (items, { client }) => {
			const account = get(currentAccountAtom)
			if (!account) {
				return null
			}

			const trashedItems = new Map<string, Set<string>>()
			for (const item of items) {
				if (item.parentId) {
					const s = trashedItems.get(item.parentId)
					if (!s) {
						trashedItems.set(item.parentId, new Set([item.id]))
					} else {
						s.add(item.id)
					}
				}
			}

			const prevDirContentMap = new Map<
				string,
				InfiniteData<typeof DirectoryContentResponse.infer> | undefined
			>()
			trashedItems.forEach((s, parentId) => {
				const key = directoryContentQueryKey(account.id, parentId)
				const prevDirContent =
					client.getQueryData<
						InfiniteData<typeof DirectoryContentResponse.infer>
					>(key)
				client.setQueryData<
					InfiniteData<typeof DirectoryContentResponse.infer>
				>(key, (prev) => {
					if (!prev) return prev
					return {
						...prev,
						pages: prev.pages.map((page) => ({
							...page,
							items: page.items.filter((it) => !s.has(it.id)),
						})),
					}
				})
				prevDirContentMap.set(parentId, prevDirContent)
			})
			return { prevDirContentMap }
		},
		onSuccess: (_data, items, _result, { client }) => {
			const account = get(currentAccountAtom)
			if (account) {
				for (const item of items) {
					if (item.parentId) {
						client.invalidateQueries({
							queryKey: directoryContentQueryKey(
								account.id,
								item.parentId,
							),
						})
					}
				}
			}
		},
		onError: (_error, _items, context, { client }) => {
			if (context) {
				const account = get(currentAccountAtom)
				if (account) {
					context.prevDirContentMap.forEach(
						(prevDirContent, parentId) => {
							client.setQueryData(
								directoryContentQueryKey(account.id, parentId),
								prevDirContent,
							)
						},
					)
				}
			}
		},
	}),
)

export const renameFileMutationAtom = atom((get) =>
	mutationOptions({
		mutationFn: async (file: FileInfo) => {
			const account = get(currentAccountAtom)
			if (!account) {
				throw new Error("Account not found")
			}

			const [, result] = await fetchApi(
				"PATCH",
				`/accounts/${account.id}/files/${file.id}`,
				{
					body: JSON.stringify({ name: file.name }),
					returns: FileInfo,
				},
			)

			return result
		},
	}),
)

export const renameDirectoryMutationAtom = atom((get) =>
	mutationOptions({
		mutationFn: async (directory: DirectoryInfo) => {
			const account = get(currentAccountAtom)
			if (!account) {
				throw new Error("Account not found")
			}

			const [, result] = await fetchApi(
				"PATCH",
				`/accounts/${account.id}/directories/${directory.id}`,
				{
					body: JSON.stringify({ name: directory.name }),
					returns: DirectoryInfo,
				},
			)

			return result
		},
		onSuccess: (data, _variables, _context, { client }) => {
			client.setQueryData(
				get(directoryInfoQueryAtom(data.id)).queryKey,
				(prev) => (prev ? { ...prev, name: data.name } : undefined),
			)
		},
	}),
)
