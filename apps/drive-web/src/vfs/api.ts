import { mutationOptions, queryOptions, skipToken } from "@tanstack/react-query"
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

export const directoryContentQueryAtom = atomFamily((directoryId: string) =>
	atom((get) => {
		const account = get(currentAccountAtom)
		return queryOptions({
			queryKey: [
				"accounts",
				account?.id,
				"directories",
				directoryId,
				"content",
			],
			queryFn: account
				? () =>
						fetchApi(
							"GET",
							`/accounts/${account.id}/directories/${directoryId}/content`,
							{ returns: DirectoryContent },
						).then(([_, result]) => result)
				: skipToken,
		})
	}),
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
			const parent = data.path.at(-2)
			if (parent) {
				client.setQueryData(
					get(directoryContentQueryAtom(parent.id)).queryKey,
					(prev) => (prev ? [...prev, data] : [data]),
				)
			}
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
			const movedItems = new Map<string, Set<string>>()

			for (const item of items) {
				if (item.parentId) {
					const s = movedItems.get(item.parentId)
					if (!s) {
						movedItems.set(item.parentId, new Set())
					} else {
						s.add(item.id)
					}
				}
			}

			const prevDirContentMap = new Map<
				string,
				DirectoryItem[] | undefined
			>()

			movedItems.forEach((s, parentId) => {
				const query = get(directoryContentQueryAtom(parentId))
				const prevDirContent = client.getQueryData(query.queryKey)
				client.setQueryData(
					query.queryKey,
					(prev) => prev?.filter((it) => !s.has(it.id)) ?? prev,
				)
				prevDirContentMap.set(parentId, prevDirContent)
			})

			return { prevDirContentMap }
		},
		onSuccess: (_data, { targetDirectory, items }, _result, { client }) => {
			const dirId =
				typeof targetDirectory === "string"
					? targetDirectory
					: targetDirectory.id
			console.log(dirId)
			client.invalidateQueries(get(directoryContentQueryAtom(dirId)))
			for (const item of items) {
				if (item.parentId) {
					client.invalidateQueries(
						get(directoryContentQueryAtom(item.parentId)),
					)
				}
			}
		},
		onError: (_error, _vars, context, { client }) => {
			if (context) {
				context.prevDirContentMap.forEach(
					(prevDirContent, parentId) => {
						client.setQueryData(
							get(directoryContentQueryAtom(parentId)).queryKey,
							prevDirContent,
						)
					},
				)
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

			const fileDeleteParams = new URLSearchParams()
			fileDeleteParams.set("id", fileIds.join(","))
			fileDeleteParams.set("trash", "true")
			const deleteFilesPromise = fetchApi(
				"DELETE",
				`/accounts/${account.id}/files?${fileDeleteParams.toString()}`,
				{
					returns: FileInfo.array(),
				},
			)

			const directoryDeleteParams = new URLSearchParams()
			directoryDeleteParams.set("id", directoryIds.join(","))
			directoryDeleteParams.set("trash", "true")
			const deleteDirectoriesPromise = fetchApi(
				"DELETE",
				`/accounts/${account.id}/directories?${directoryDeleteParams.toString()}`,
				{
					returns: DirectoryInfo.array(),
				},
			)

			const [[, deletedFiles], [, deletedDirectories]] =
				await Promise.all([
					deleteFilesPromise,
					deleteDirectoriesPromise,
				])

			return [...deletedFiles, ...deletedDirectories]
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
