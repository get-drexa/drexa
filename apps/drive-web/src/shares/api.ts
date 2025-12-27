import { infiniteQueryOptions, queryOptions } from "@tanstack/react-query"
import { fetchApi } from "@/lib/api"
import {
	type DirectoryContentOrderBy,
	type DirectoryContentOrderDirection,
	DirectoryContentResponse,
} from "@/vfs/api"
import { DirectoryInfoWithPath } from "@/vfs/vfs"

const baseApiUrl = new URL(
	import.meta.env.VITE_API_URL ??
		`${location.protocol}//${location.host}/api`,
)

function buildShareApiUrl(path: string): URL {
	const basePath = baseApiUrl.pathname.endsWith("/")
		? baseApiUrl.pathname.slice(0, -1)
		: baseApiUrl.pathname
	return new URL(`${basePath}${path}`, baseApiUrl)
}

function buildQueryString(params: Record<string, string | undefined>): string {
	const searchParams = new URLSearchParams()
	for (const [key, value] of Object.entries(params)) {
		if (value) {
			searchParams.set(key, value)
		}
	}
	const queryString = searchParams.toString()
	return queryString ? `?${queryString}` : ""
}

type ShareDirectoryInfoQueryParams = {
	shareId: string
	directoryId: string
	accountId?: string
}

export const shareDirectoryInfoQueryKey = (
	shareId: string,
	directoryId: string,
	accountId?: string,
): readonly unknown[] => [
	"shares",
	shareId,
	"directories",
	directoryId,
	"info",
	accountId ?? "public",
]

export function shareDirectoryInfoQuery({
	shareId,
	directoryId,
	accountId,
}: ShareDirectoryInfoQueryParams) {
	const queryString = buildQueryString({
		include: "path",
		accountId,
	})

	return queryOptions({
		queryKey: shareDirectoryInfoQueryKey(shareId, directoryId, accountId),
		queryFn: () =>
			fetchApi(
				"GET",
				`/shares/${shareId}/directories/${directoryId}${queryString}`,
				{ returns: DirectoryInfoWithPath },
			).then(([_, result]) => result),
	})
}

type ShareDirectoryContentQueryParams = {
	shareId: string
	directoryId: string
	orderBy: DirectoryContentOrderBy
	direction: DirectoryContentOrderDirection
	limit: number
	accountId?: string
}

export const shareDirectoryContentQueryKey = (
	shareId: string,
	directoryId: string,
	params?: {
		orderBy?: DirectoryContentOrderBy
		direction?: DirectoryContentOrderDirection
		accountId?: string
	},
): readonly unknown[] => [
	"shares",
	shareId,
	"directories",
	directoryId,
	"content",
	...(params
		? [
				{
					orderBy: params.orderBy,
					direction: params.direction,
					accountId: params.accountId,
				},
			]
		: []),
]

type ShareDirectoryContentPageParam = {
	orderBy: DirectoryContentOrderBy
	direction: DirectoryContentOrderDirection
	limit: number
	cursor: string
}

export function shareDirectoryContentQuery({
	shareId,
	directoryId,
	orderBy,
	direction,
	limit,
	accountId,
}: ShareDirectoryContentQueryParams) {
	return infiniteQueryOptions({
		queryKey: shareDirectoryContentQueryKey(shareId, directoryId, {
			orderBy,
			direction,
			accountId,
		}),
		initialPageParam: {
			orderBy,
			direction,
			limit,
			cursor: "",
		} satisfies ShareDirectoryContentPageParam,
		queryFn: ({ pageParam }) => {
			const queryString = buildQueryString({
				orderBy: pageParam.orderBy,
				dir: pageParam.direction,
				limit: String(pageParam.limit),
				cursor: pageParam.cursor || undefined,
				accountId,
			})
			return fetchApi(
				"GET",
				`/shares/${shareId}/directories/${directoryId}/content${queryString}`,
				{ returns: DirectoryContentResponse },
			).then(([_, result]) => result)
		},
		getNextPageParam: (lastPage, _pages, lastPageParam) =>
			lastPage.nextCursor
				? {
						...lastPageParam,
						cursor: lastPage.nextCursor,
					}
				: null,
	})
}

type ShareFileContentUrlParams = {
	shareId: string
	fileId: string
	accountId?: string
}

export function shareFileContentUrl({
	shareId,
	fileId,
	accountId,
}: ShareFileContentUrlParams): string {
	const url = buildShareApiUrl(`/shares/${shareId}/files/${fileId}/content`)
	if (accountId) {
		url.searchParams.set("accountId", accountId)
	}
	return url.toString()
}
