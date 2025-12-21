import type { InfiniteData, QueryClient } from "@tanstack/react-query"
import type { DirectoryContentResponseType } from "@/vfs/api"
import type { DirectoryItem } from "@/vfs/vfs"

export type DirectoryContentOptimisticUpdate = {
	queryKey: readonly unknown[]
	prevDirContent: InfiniteData<DirectoryContentResponseType> | undefined
}

type OptimisticRemovalParams = {
	queryKey: readonly unknown[]
	items: DirectoryItem[]
}

export function optimisticallyRemoveDirectoryItems(
	client: QueryClient,
	{ queryKey, items }: OptimisticRemovalParams,
): DirectoryContentOptimisticUpdate {
	const prevDirContent =
		client.getQueryData<InfiniteData<DirectoryContentResponseType>>(
			queryKey,
		)
	const removedItemIds = new Set(items.map((item) => item.id))

	client.setQueryData<InfiniteData<DirectoryContentResponseType>>(
		queryKey,
		(prev) => {
			if (!prev) return prev
			return {
				...prev,
				pages: prev.pages.map((page) => ({
					...page,
					items: page.items.filter(
						(item) => !removedItemIds.has(item.id),
					),
				})),
			}
		},
	)

	return { queryKey, prevDirContent }
}

export function rollbackDirectoryContentOptimisticUpdate(
	client: QueryClient,
	update?: DirectoryContentOptimisticUpdate | null,
) {
	if (update) {
		client.setQueryData(update.queryKey, update.prevDirContent)
	}
}
