import {
	type InfiniteData,
	infiniteQueryOptions,
	mutationOptions,
	queryOptions,
	skipToken,
} from "@tanstack/react-query"
import { type } from "arktype"
import { type Drive } from "@/drive/drive"
import { fetchApi } from "@/lib/api"
import type { Organization } from "@/organization/organization"
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

export const rootDirectoryQuery = ({
	org,
	drive,
}: {
	org: Organization | null
	drive: Drive | null
}) =>
	queryOptions({
		queryKey: ["organizations", org?.slug, "drives", drive?.id, "root"],
		queryFn:
			org && drive
				? () =>
						fetchApi(
							"GET",
							`/${org.slug}/drives/${drive.id}/directories/root?include=path`,
							{ returns: DirectoryInfoWithPath },
						).then(([_, result]) => result)
				: skipToken,
	})

export const directoryInfoQuery = ({
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
			"directories",
			directoryId,
		],
		queryFn:
			org && drive
				? () =>
						fetchApi(
							"GET",
							`/${org.slug}/drives/${drive.id}/directories/${directoryId}?include=path`,
							{ returns: DirectoryInfoWithPath },
						).then(([_, result]) => result)
				: skipToken,
	})

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
	orgSlug: string | undefined,
	driveId: string | undefined,
	directoryId: string,
	params?: {
		orderBy?: DirectoryContentOrderBy
		direction?: DirectoryContentOrderDirection
	},
): readonly unknown[] => [
	"organizations",
	orgSlug,
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

export const directoryContentQuery = ({
	org,
	drive,
	directoryId,
	orderBy,
	direction,
	limit,
}: {
	org: Organization | null
	drive: Drive | null
} & DirectoryContentQueryParams) =>
	infiniteQueryOptions({
		queryKey: directoryContentQueryKey(org?.slug, drive?.id, directoryId, {
			orderBy,
			direction,
		}),
		initialPageParam: {
			orderBy,
			direction,
			limit,
			cursor: "",
		} satisfies DirectoryContentPageParam,
		queryFn:
			org && drive
				? ({ pageParam }) =>
						fetchApi(
							"GET",
							`/${org.slug}/drives/${drive.id}/directories/${directoryId}/content?orderBy=${pageParam.orderBy}&dir=${pageParam.direction}&limit=${pageParam.limit}${pageParam.cursor ? `&cursor=${pageParam.cursor}` : ""}`,
							{ returns: DirectoryContentResponse },
						).then(([_, result]) => result)
				: skipToken,
		getNextPageParam: (lastPage, _pages, lastPageParam) =>
			lastPage.nextCursor
				? {
						...lastPageParam,
						cursor: lastPage.nextCursor,
					}
				: null,
		})

export const createDirectoryMutationOptions = ({
	org,
	drive,
}: {
	org: Organization
	drive: Drive | null
}) =>
	mutationOptions({
		mutationFn: async (data: { name: string; parentId: string }) => {
			if (!drive) throw new Error("No drive selected")
			return fetchApi(
				"POST",
				`/${org.slug}/drives/${drive.id}/directories?include=path`,
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
			if (!drive) return
			client.invalidateQueries({
				queryKey: directoryContentQueryKey(org.slug, drive.id, parentId),
			})
		},
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

export const moveDirectoryItemsMutationOptions = ({
	org,
	drive,
}: {
	org: Organization
	drive: Drive | null
}) =>
	mutationOptions({
		mutationFn: async ({
			targetDirectory,
			items,
			}: {
			targetDirectory: DirectoryInfo | string
			items: DirectoryItem[]
		}) => {
			if (!drive) {
				throw new Error("Drive not found")
			}

			const dirId =
				typeof targetDirectory === "string"
					? targetDirectory
					: targetDirectory.id

			const [, result] = await fetchApi(
				"POST",
				`/${org.slug}/drives/${drive.id}/directories/${dirId}/content`,
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
			if (!drive) return

			const dirId =
				typeof targetDirectory === "string"
					? targetDirectory
					: targetDirectory.id
			// Invalidate using base key (without params) to invalidate all queries for these directories
			client.invalidateQueries({
				queryKey: directoryContentQueryKey(org.slug, drive.id, dirId),
			})
			for (const item of items) {
				if (item.parentId) {
					client.invalidateQueries({
						queryKey: directoryContentQueryKey(
							org.slug,
							drive.id,
							item.parentId,
						),
					})
				}
			}
		},
	})

export const moveToTrashMutationOptions = ({
	org,
	drive,
}: {
	org: Organization
	drive: Drive | null
}) =>
	mutationOptions({
		mutationFn: async (items: DirectoryItem[]) => {
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
					`/${org.slug}/drives/${drive.id}/files?${fileDeleteParams.toString()}`,
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
					`/${org.slug}/drives/${drive.id}/directories?${directoryDeleteParams.toString()}`,
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
			if (!drive) return
			// Invalidate using base key (without params) to invalidate all queries for these directories
			for (const item of items) {
				if (item.parentId) {
					client.invalidateQueries({
						queryKey: directoryContentQueryKey(
							org.slug,
							drive.id,
							item.parentId,
						),
					})
				}
			}
		},
	})

export const renameFileMutationOptions = ({
	org,
	drive,
}: {
	org: Organization
	drive: Drive | null
}) =>
	mutationOptions({
		mutationFn: async (file: FileInfo) => {
			if (!drive) {
				throw new Error("Drive not found")
			}

			const [, result] = await fetchApi(
				"PATCH",
				`/${org.slug}/drives/${drive.id}/files/${file.id}`,
				{
					body: JSON.stringify({ name: file.name }),
					returns: FileInfo,
				},
			)

			return result
		},
	})

export const renameDirectoryMutationOptions = ({
	org,
	drive,
}: {
	org: Organization
	drive: Drive | null
}) =>
	mutationOptions({
		mutationFn: async (directory: DirectoryInfo) => {
			if (!drive) {
				throw new Error("Drive not found")
			}

			const [, result] = await fetchApi(
				"PATCH",
				`/${org.slug}/drives/${drive.id}/directories/${directory.id}`,
				{
					body: JSON.stringify({ name: directory.name }),
					returns: DirectoryInfo,
				},
			)

			return result
		},
		onSuccess: (data, _variables, _context, { client }) => {
			if (!drive) return
			client.setQueryData(
				directoryInfoQuery({ org, drive, directoryId: data.id })
					.queryKey,
				(prev) => (prev ? { ...prev, name: data.name } : undefined),
			)
		},
	})
