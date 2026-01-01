import { useQuery } from "@tanstack/react-query"
import { createFileRoute } from "@tanstack/react-router"
import { type } from "arktype"
import { atom } from "jotai"
import { useCallback, useMemo } from "react"
import { DirectoryPageContext } from "@/directories/directory-page/context"
import {
	DEFAULT_DIRECTORY_CONTENT_ORDER_BY,
	DEFAULT_DIRECTORY_CONTENT_ORDER_DIRECTION,
} from "@/directories/directory-page/defaults"
import { DirectoryContentTable } from "@/directories/directory-page/directory-content-table"
import { DirectoryPageSkeleton } from "@/directories/directory-page/directory-page-skeleton"
import { DirectoryPathBreadcrumb } from "@/directories/directory-path-breadcrumb"
import type { FileDragInfo } from "@/files/use-file-drop"
import {
	shareDirectoryContentQuery,
	shareDirectoryInfoQuery,
	shareFileContentUrl,
} from "@/shares/api"
import {
	DIRECTORY_CONTENT_ORDER_BY,
	DIRECTORY_CONTENT_ORDER_DIRECTION,
	type DirectoryContentOrderBy,
	type DirectoryContentOrderDirection,
} from "@/vfs/api"
import type { DirectoryInfo, FileInfo } from "@/vfs/vfs"

const DirectoryContentPageParams = type({
	orderBy: type
		.valueOf(DIRECTORY_CONTENT_ORDER_BY)
		.default(DEFAULT_DIRECTORY_CONTENT_ORDER_BY),
	direction: type
		.valueOf(DIRECTORY_CONTENT_ORDER_DIRECTION)
		.default(DEFAULT_DIRECTORY_CONTENT_ORDER_DIRECTION),
})

const fileDragInfoAtom = atom<FileDragInfo | null>(null)

export const Route = createFileRoute(
	"/shares/$shareId/directories/$directoryId",
)({
	validateSearch: DirectoryContentPageParams,
	component: RouteComponent,
})

function RouteComponent() {
	const { shareId, directoryId } = Route.useParams()
	const search = Route.useSearch()
	const navigate = Route.useNavigate()

	const { data: directoryInfo, isLoading: isLoadingDirectoryInfo } = useQuery(
		shareDirectoryInfoQuery({ shareId, directoryId }),
	)

	const directoryUrlById = useCallback(
		(targetDirectoryId: string) =>
			`/shares/${shareId}/directories/${targetDirectoryId}`,
		[shareId],
	)

	const directoryUrlFn = useCallback(
		(directory: DirectoryInfo) => directoryUrlById(directory.id),
		[directoryUrlById],
	)

	const query = useMemo(
		() =>
			shareDirectoryContentQuery({
				shareId,
				directoryId,
				orderBy: search.orderBy,
				direction: search.direction,
				limit: 100,
			}),
		[shareId, directoryId, search.orderBy, search.direction],
	)

	const applySorting = useCallback(
		(
			orderBy: DirectoryContentOrderBy,
			direction: DirectoryContentOrderDirection,
		) => {
			navigate({
				search: {
					orderBy,
					direction,
				},
			})
		},
		[navigate],
	)

	const onTableOpenFile = useCallback(
		(file: FileInfo) => {
			const url = shareFileContentUrl({
				shareId,
				fileId: file.id,
			})
			window.open(url, "_blank", "noopener,noreferrer")
		},
		[shareId],
	)

	if (isLoadingDirectoryInfo) {
		return <DirectoryPageSkeleton />
	}

	if (!directoryInfo) {
		return null
	}

	return (
		<DirectoryPageContext value={{ directory: directoryInfo }}>
			<header className="flex py-2 shrink-0 items-center gap-2 border-b px-4 w-full">
				<DirectoryPathBreadcrumb
					directory={directoryInfo}
					rootLabel="Shared"
					directoryUrlFn={directoryUrlById}
				/>
			</header>
			<div className="w-full min-h-0">
				<DirectoryContentTable
					query={query}
					directoryUrlFn={directoryUrlFn}
					fileDragInfoAtom={fileDragInfoAtom}
					onContextMenu={() => {}}
					onOpenFile={onTableOpenFile}
					onSortChange={applySorting}
					readOnly
				/>
			</div>
		</DirectoryPageContext>
	)
}
