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
import { fetchApi } from "@/lib/api"
import { currentDriveAtom } from "@/drive/drive"
import {
	DirectoryContent,
	DirectoryInfo,
	DirectoryInfoWithPath,
	DirectoryItem,
	FileInfo,
} from "./vfs"

export const DirectoryContentResponse = type({
	items: DirectoryContent,
	"nextCursor?": "string",
})
export type DirectoryContentResponseType = typeof DirectoryContentResponse.infer

/**
 * This atom derives the file url for a given file.
 * It is recommended to use {@link useFileUrl} instead of using this atom directly.
 */
export const fileUrlAtom = atomFamily((fileId: string) =>
	atom((get) => {
		const drive = get(currentDriveAtom)
		if (!drive) {
			return ""
		}
		return `${import.meta.env.VITE_API_URL}/drives/${drive.id}/files/${fileId}/content`
	}),
)

export const rootDirectoryQueryAtom = atom((get) => {
	const drive = get(currentDriveAtom)
	return queryOptions({
		queryKey: ["drives", drive?.id, "directories", "root"],
		queryFn: drive
			? () =>
					fetchApi(
						"GET",
						`/drives/${drive.id}/directories/root?include=path`,
						{ returns: DirectoryInfoWithPath },
					).then(([_, result]) => result)
			: skipToken,
	})
})

export const directoryInfoQueryAtom = atomFamily((directoryId: string) =>
	atom((get) => {
		const drive = get(currentDriveAtom)
		return queryOptions({
			queryKey: ["drives", drive?.id, "directories", directoryId],
			queryFn: drive
				? () =>
						fetchApi(
							"GET",
							`/drives/${drive.id}/directories/${directoryId}?include=path`,
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
export type DirectoryContentOrderDirection =
	(typeof DIRECTORY_CONTENT_ORDER_DIRECTION)[keyof typeof DIRECTORY_CONTENT_ORDER_DIRECTION]

type DirectoryContentQueryParams = {
	directoryId: string
	orderBy: DirectoryContentOrderBy
	direction: DirectoryContentOrderDirection
	limit: number
}

type DirectoryContentPageParam = {
	orderBy: DirectoryContentOrderBy
	direction: DirectoryContentOrderDirection
	limit: number
	cursor: string
}

export const directoryContentQueryKey = (
	driveId: string | undefined,
	directoryId: string,
	params?: {
		orderBy?: DirectoryContentOrderBy
		direction?: DirectoryContentOrderDirection
	},
): readonly unknown[] => [
	"drives",
	driveId,
	"directories",
	directoryId,
	"content",
	...(params ? [{ orderBy: params.orderBy, dir: params.direction }] : []),
]
export type DirectoryContentQuery = ReturnType<
	typeof infiniteQueryOptions<
		typeof DirectoryContentResponse.infer,
		Error,
		InfiniteData<typeof DirectoryContentResponse.infer>,
		readonly unknown[],
		DirectoryContentPageParam
	>
>
export const directoryContentQueryAtom = atomFamily(
	({ directoryId, orderBy, direction, limit }: DirectoryContentQueryParams) =>
		atom((get) => {
			const drive = get(currentDriveAtom)
			return infiniteQueryOptions({
				queryKey: directoryContentQueryKey(drive?.id, directoryId, {
					orderBy,
					direction,
				}),
				initialPageParam: {
					orderBy,
					direction,
					limit,
					cursor: "",
				},
				queryFn: ({ pageParam }) =>
					drive
						? fetchApi(
								"GET",
								`/drives/${drive.id}/directories/${directoryId}/content?orderBy=${pageParam.orderBy}&dir=${pageParam.direction}&limit=${pageParam.limit}${pageParam.cursor ? `&cursor=${pageParam.cursor}` : ""}`,
								{ returns: DirectoryContentResponse },
							).then(([_, result]) => result)
						: Promise.reject(new Error("No drive selected")),
				getNextPageParam: (lastPage, _pages, lastPageParam) =>
					lastPage.nextCursor
						? {
								...lastPageParam,
								cursor: lastPage.nextCursor,
							}
						: null,
			})
		}),
	(paramsA, paramsB) =>
		paramsA.directoryId === paramsB.directoryId &&
		paramsA.orderBy === paramsB.orderBy &&
		paramsA.direction === paramsB.direction &&
		paramsA.limit === paramsB.limit,
)

export const createDirectoryMutationAtom = atom((get) => {
	const drive = get(currentDriveAtom)
	return mutationOptions({
		mutationFn: async (data: { name: string; parentId: string }) => {
			if (!drive) throw new Error("No drive selected")
			return fetchApi(
				"POST",
				`/drives/${drive.id}/directories?include=path`,
				{
					body: JSON.stringify({
						name: data.name,
						parentId: data.parentId,
					}),
					returns: DirectoryInfoWithPath,
				},
			).then(([_, result]) => result)
		},
		onSuccess: (_data, { parentId }, _context, { client }) => {
			if (drive) {
				client.invalidateQueries({
					queryKey: directoryContentQueryKey(drive.id, parentId),
				})
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
			const drive = get(currentDriveAtom)
			if (!drive) {
				throw new Error("Drive not found")
			}

			const dirId =
				typeof targetDirectory === "string"
					? targetDirectory
					: targetDirectory.id

			const [, result] = await fetchApi(
				"POST",
				`/drives/${drive.id}/directories/${dirId}/content`,
				{
					body: JSON.stringify({
						items: items.map((item) => item.id),
					}),
					returns: MoveDirectoryItemsResult,
				},
			)
			return result
		},
		onSuccess: (_data, { targetDirectory, items }, _result, { client }) => {
			const drive = get(currentDriveAtom)
			if (!drive) return

			const dirId =
				typeof targetDirectory === "string"
					? targetDirectory
					: targetDirectory.id
			// Invalidate using base key (without params) to invalidate all queries for these directories
			client.invalidateQueries({
				queryKey: directoryContentQueryKey(drive.id, dirId),
			})
			for (const item of items) {
				if (item.parentId) {
					client.invalidateQueries({
						queryKey: directoryContentQueryKey(
							drive.id,
							item.parentId,
						),
					})
				}
			}
		},
	}),
)

export const moveToTrashMutationAtom = atom((get) =>
	mutationOptions({
		mutationFn: async (items: DirectoryItem[]) => {
			const drive = get(currentDriveAtom)
			if (!drive) {
				throw new Error("Drive not found")
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
					`/drives/${drive.id}/files?${fileDeleteParams.toString()}`,
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
					`/drives/${drive.id}/directories?${directoryDeleteParams.toString()}`,
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
		onSuccess: (_data, items, _result, { client }) => {
			const drive = get(currentDriveAtom)
			if (drive) {
				// Invalidate using base key (without params) to invalidate all queries for these directories
				for (const item of items) {
					if (item.parentId) {
						client.invalidateQueries({
							queryKey: directoryContentQueryKey(
								drive.id,
								item.parentId,
							),
						})
					}
				}
			}
		},
	}),
)

export const renameFileMutationAtom = atom((get) =>
	mutationOptions({
		mutationFn: async (file: FileInfo) => {
			const drive = get(currentDriveAtom)
			if (!drive) {
				throw new Error("Drive not found")
			}

			const [, result] = await fetchApi(
				"PATCH",
				`/drives/${drive.id}/files/${file.id}`,
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
			const drive = get(currentDriveAtom)
			if (!drive) {
				throw new Error("Drive not found")
			}

			const [, result] = await fetchApi(
				"PATCH",
				`/drives/${drive.id}/directories/${directory.id}`,
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
