import { useMutation, useQuery } from "@tanstack/react-query"
import { createFileRoute } from "@tanstack/react-router"
import type { Row, Table } from "@tanstack/react-table"
import { type } from "arktype"
import { atom, useAtom, useAtomValue, useSetAtom, useStore } from "jotai"
import { atomWithStorage } from "jotai/utils"
import {
	ChevronDownIcon,
	PlusIcon,
	ScissorsIcon,
	Share2Icon,
	TextCursorInputIcon,
	TrashIcon,
} from "lucide-react"
import { lazy, Suspense, useCallback, useContext } from "react"
import { toast } from "sonner"
import { currentDriveAtom } from "@/drive/drive"
import { useCurrentOrganization } from "@/organization/context"
import { DirectoryIcon } from "@/components/icons/directory-icon"
import { TextFileIcon } from "@/components/icons/text-file-icon"
import { Button } from "@/components/ui/button"
import {
	ContextMenu,
	ContextMenuContent,
	ContextMenuItem,
	ContextMenuTrigger,
} from "@/components/ui/context-menu"
import {
	DropdownMenu,
	DropdownMenuContent,
	DropdownMenuItem,
	DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { WithAtom } from "@/components/with-atom"
import { backgroundTaskProgressAtom } from "@/dashboard/state"
import { DirectoryPageContext } from "@/directories/directory-page/context"
import {
	DEFAULT_DIRECTORY_CONTENT_ORDER_BY,
	DEFAULT_DIRECTORY_CONTENT_ORDER_DIRECTION,
} from "@/directories/directory-page/defaults"
import { DirectoryContentTable } from "@/directories/directory-page/directory-content-table"
import { DirectoryPageSkeleton } from "@/directories/directory-page/directory-page-skeleton"
import { NewDirectoryDialog } from "@/directories/directory-page/new-directory-dialog"
import { RenameFileDialog } from "@/directories/directory-page/rename-file-dialog"
import { DirectoryPathBreadcrumb } from "@/directories/directory-path-breadcrumb"
import { FilePreviewDialog } from "@/files/file-preview-dialog"
import { cutItemsAtom, inProgressFileUploadCountAtom } from "@/files/store"
import { UploadFileDialog } from "@/files/upload-file-dialog"
import type { FileDragInfo } from "@/files/use-file-drop"
import { formatError } from "@/lib/error"
import type {
	DirectoryContentOrderBy,
	DirectoryContentOrderDirection,
} from "@/vfs/api"
import {
	DIRECTORY_CONTENT_ORDER_BY,
	DIRECTORY_CONTENT_ORDER_DIRECTION,
	directoryContentQuery,
	directoryContentQueryKey,
	directoryInfoQuery,
	moveToTrashMutationOptions,
} from "@/vfs/api"
import {
	optimisticallyRemoveDirectoryItems,
	rollbackDirectoryContentOptimisticUpdate,
} from "@/vfs/optimistic"
import type {
	DirectoryInfo,
	DirectoryInfoWithPath,
	DirectoryItem,
	FileInfo,
} from "@/vfs/vfs"
import { ItemShareDialog } from "@/sharing/item-share-dialog"

// Conditional lazy import - Vite will tree-shake this entire import in production
// because import.meta.env.DEV is evaluated at build time
const MockDirectoryContentTable = import.meta.env.DEV
	? lazy(() =>
			import(
				"@/directories/directory-page/mock-directory-content-table"
			).then((mod) => ({
				default: mod.MockDirectoryContentTable,
			})),
		)
	: null

const DirectoryContentPageParams = type({
	orderBy: type
		.valueOf(DIRECTORY_CONTENT_ORDER_BY)
		.default(DEFAULT_DIRECTORY_CONTENT_ORDER_BY),
	direction: type
		.valueOf(DIRECTORY_CONTENT_ORDER_DIRECTION)
		.default(DEFAULT_DIRECTORY_CONTENT_ORDER_DIRECTION),
})

export const Route = createFileRoute(
	"/_authenticated/$orgSlug/_sidebar-layout/directories/$directoryId",
)({
	validateSearch: DirectoryContentPageParams,
	component: RouteComponent,
})

enum DialogKind {
	NewDirectory = "NewDirectory",
	UploadFile = "UploadFile",
	ItemShare = "ItemShare",
}

type NewDirectoryDialogData = {
	kind: DialogKind.NewDirectory
}

type UploadFileDialogData = {
	kind: DialogKind.UploadFile
	directory: DirectoryInfoWithPath
}

type ItemShareDialogData = {
	kind: DialogKind.ItemShare
	item: DirectoryItem
}

type ActiveDialogData =
	| NewDirectoryDialogData
	| UploadFileDialogData
	| ItemShareDialogData

// MARK: atoms
const contextMenuTargetItemsAtom = atom<DirectoryItem[]>([])
const activeDialogDataAtom = atom<ActiveDialogData | null>(null)
const fileDragInfoAtom = atom<FileDragInfo | null>(null)
const openedFileAtom = atom<FileInfo | null>(null)
const itemBeingRenamedAtom = atom<{
	originalItem: DirectoryItem
	name: string
} | null>(null)

// Atom to control mock table usage
// In dev mode: uses atomWithStorage to persist in localStorage
// In production: uses regular atom (always false, tree-shaken)
// This ensures hooks are always called in the same order
const mockTableAtom = import.meta.env.DEV
	? atomWithStorage("drexa:use-mock-directory-table", false)
	: atom(false)

// MARK: page entry
function RouteComponent() {
	const { directoryId, orgSlug } = Route.useParams()
	const drive = useAtomValue(currentDriveAtom)
	const org = useCurrentOrganization()
	const { data: directoryInfo, isLoading: isLoadingDirectoryInfo } = useQuery(
		directoryInfoQuery({ org, drive, directoryId }),
	)

	const directoryUrlById = useCallback(
		(directoryId: string) => `/${orgSlug}/directories/${directoryId}`,
		[orgSlug],
	)

	if (isLoadingDirectoryInfo) {
		return <DirectoryPageSkeleton />
	}

	if (!directoryInfo) {
		// TODO: handle empty state/error
		return null
	}

	return (
		<DirectoryPageContext value={{ directory: directoryInfo }}>
			<header className="flex py-2 shrink-0 items-center gap-2 border-b px-4 w-full">
				<DirectoryPathBreadcrumb
					directory={directoryInfo}
					rootLabel="All Files"
					directoryUrlFn={directoryUrlById}
					fileDragInfoAtom={fileDragInfoAtom}
				/>
				<div className="ml-auto flex flex-row gap-2">
					{import.meta.env.DEV && <MockTableToggle />}
					<NewDirectoryItemDropdown />
					<UploadFileButton />
				</div>
			</header>

			{/* DirectoryContentContextMenu must wrap div instead of DirectoryContentTable, otherwise radix will throw "event.preventDefault is not a function" error, idk why */}
			<DirectoryContentContextMenu>
				<div className="w-full min-h-0">
					<_DirectoryContentTable />
				</div>
			</DirectoryContentContextMenu>

			<WithAtom atom={activeDialogDataAtom}>
				{(data, setData) => (
					<>
						<NewDirectoryDialog
							open={data?.kind === DialogKind.NewDirectory}
							parentDirectory={directoryInfo}
							onOpenChange={(open) => {
								if (!open) {
									setData(null)
								}
							}}
						/>
						{data?.kind === DialogKind.UploadFile && (
							<UploadFileDialog
								targetDirectory={directoryInfo}
								onClose={() => setData(null)}
							/>
						)}
					</>
				)}
			</WithAtom>

			<WithAtom atom={activeDialogDataAtom}>
				{(data, setData) => {
					return (
						<ItemShareDialog
							item={
								data?.kind === DialogKind.ItemShare
									? data.item
									: null
							}
							open={data?.kind === DialogKind.ItemShare}
							onClose={() => setData(null)}
						/>
					)
				}}
			</WithAtom>

			<WithAtom atom={itemBeingRenamedAtom}>
				{(itemBeingRenamed, setItemBeingRenamed) => {
					if (!itemBeingRenamed) return null
					return (
						<RenameFileDialog
							item={itemBeingRenamed.originalItem}
							onRenameSuccess={() => {
								toast.success("File renamed successfully")
								setItemBeingRenamed(null)
							}}
							onClose={() => setItemBeingRenamed(null)}
						/>
					)
				}}
			</WithAtom>

			<WithAtom atom={openedFileAtom}>
				{(openedFile, setOpenedFile) => {
					if (!openedFile) return null
					return (
						<FilePreviewDialog
							openedFile={openedFile}
							onClose={() => setOpenedFile(null)}
						/>
					)
				}}
			</WithAtom>
		</DirectoryPageContext>
	)
}

function _DirectoryContentTable() {
	// Always call the hook - in production the atom always returns false
	const useMock = useAtomValue(mockTableAtom)
	const { directory } = useContext(DirectoryPageContext)
	const navigate = Route.useNavigate()
	const drive = useAtomValue(currentDriveAtom)
	const org = useCurrentOrganization()

	const search = Route.useSearch()
	const query = directoryContentQuery({
		org,
		drive,
		directoryId: directory.id,
		orderBy: search.orderBy,
		direction: search.direction,
		limit: 100,
	})

	const setOpenedFile = useSetAtom(openedFileAtom)
	const setContextMenuTargetItems = useSetAtom(contextMenuTargetItemsAtom)

	const onTableOpenFile = useCallback(
		(file: FileInfo) => {
			setOpenedFile(file)
		},
		[setOpenedFile],
	)

	const { orgSlug } = Route.useParams()
	const directoryUrlFn = useCallback(
		(directory: DirectoryInfo) => `/${orgSlug}/directories/${directory.id}`,
		[orgSlug],
	)

	const handleContextMenuRequest = useCallback(
		(row: Row<DirectoryItem>, table: Table<DirectoryItem>) => {
			if (row.getIsSelected()) {
				setContextMenuTargetItems(
					table.getSelectedRowModel().rows.map((row) => row.original),
				)
			} else {
				setContextMenuTargetItems([row.original])
			}
		},
		[setContextMenuTargetItems],
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

	// In production, MockDirectoryContentTable is null, so this always renders real table
	if (import.meta.env.DEV && useMock && MockDirectoryContentTable) {
		return (
			<Suspense fallback={<div>Loading mock table...</div>}>
				<MockDirectoryContentTable
					directoryUrlFn={directoryUrlFn}
					onContextMenu={handleContextMenuRequest}
					onOpenFile={onTableOpenFile}
					query={query}
				/>
			</Suspense>
		)
	}

	return (
		<DirectoryContentTable
			query={query}
			directoryUrlFn={directoryUrlFn}
			fileDragInfoAtom={fileDragInfoAtom}
			onContextMenu={handleContextMenuRequest}
			onOpenFile={onTableOpenFile}
			onSortChange={applySorting}
		/>
	)
}

function MockTableToggle() {
	// Always call the hook - mockTableAtom is always defined
	// (it's a regular atom in production, but we only render in dev mode)
	const [isEnabled, setIsEnabled] = useAtom(mockTableAtom)

	if (!import.meta.env.DEV) {
		return null
	}

	const handleToggle = () => {
		setIsEnabled((prev) => !prev)
	}

	return (
		<Button
			variant={isEnabled ? "default" : "outline"}
			size="sm"
			onClick={handleToggle}
			title="Toggle mock directory table (dev only)"
		>
			{isEnabled ? "Mock: ON" : "Mock: OFF"}
		</Button>
	)
}

// ==================================
// MARK: ctx menu

// tags: ctxmenu contextmenu directorycontextmenu
function DirectoryContentContextMenu({
	children,
}: {
	children: React.ReactNode
}) {
	const store = useStore()
	const [target, setTarget] = useAtom(contextMenuTargetItemsAtom)
	const setBackgroundTaskProgress = useSetAtom(backgroundTaskProgressAtom)
	const setCutItems = useSetAtom(cutItemsAtom)
	const drive = useAtomValue(currentDriveAtom)
	const org = useCurrentOrganization()
	const { directory } = useContext(DirectoryPageContext)
	const search = Route.useSearch()
	const setActiveDialogData = useSetAtom(activeDialogDataAtom)

	const moveToTrashMutation = moveToTrashMutationOptions({ org, drive })
	const { mutate: moveToTrash } = useMutation({
		...moveToTrashMutation,
		onMutate: (items, { client }) => {
			setBackgroundTaskProgress({
				label: "Moving items to trash…",
			})
			if (!drive) {
				return null
			}
			return optimisticallyRemoveDirectoryItems(client, {
				queryKey: directoryContentQueryKey(org.slug, drive.id, directory.id, {
					orderBy: search.orderBy,
					direction: search.direction,
				}),
				items,
			})
		},
		onSuccess: (data) => {
			setBackgroundTaskProgress(null)
			toast.success(`Moved ${data.length} items to trash`)
		},
		onError: (err, _vars, mutateResult, { client }) => {
			toast.error(formatError(err))
			setBackgroundTaskProgress(null)
			rollbackDirectoryContentOptimisticUpdate(client, mutateResult)
		},
	})

	const handleCut = () => {
		const selectedItems = store.get(contextMenuTargetItemsAtom)
		if (selectedItems.length > 0) {
			setCutItems(selectedItems)
		}
	}

	const handleDelete = () => {
		const selectedItems = store.get(contextMenuTargetItemsAtom)
		if (selectedItems.length > 0) {
			moveToTrash(selectedItems)
		}
	}

	const openShareDialog = () => {
		const selectedItems = store.get(contextMenuTargetItemsAtom)
		setActiveDialogData({
			kind: DialogKind.ItemShare,
			item: selectedItems[0]!,
		})
	}

	return (
		<ContextMenu
			onOpenChange={(open) => {
				if (!open) {
					setTarget([])
				}
			}}
		>
			<ContextMenuTrigger asChild>{children}</ContextMenuTrigger>
			{target.length > 0 && (
				<ContextMenuContent>
					<RenameMenuItem />
					<ContextMenuItem onClick={handleCut}>
						<ScissorsIcon />
						Cut
					</ContextMenuItem>
					<ContextMenuItem onClick={openShareDialog}>
						<Share2Icon />
						Share
					</ContextMenuItem>
					<ContextMenuItem
						variant="destructive"
						onClick={handleDelete}
					>
						<TrashIcon />
						Move to trash
					</ContextMenuItem>
				</ContextMenuContent>
			)}
		</ContextMenu>
	)
}

function RenameMenuItem() {
	const store = useStore()
	const target = useAtomValue(contextMenuTargetItemsAtom)
	const setItemBeingRenamed = useSetAtom(itemBeingRenamedAtom)

	const handleRename = () => {
		const selectedItems = store.get(contextMenuTargetItemsAtom)
		if (selectedItems.length === 1) {
			// biome-ignore lint/style/noNonNullAssertion: length is checked
			const selectedItem = selectedItems[0]!
			setItemBeingRenamed({
				originalItem: selectedItem,
				name: selectedItem.name,
			})
		}
	}

	// Only render if exactly one item is selected
	if (target.length !== 1) {
		return null
	}

	return (
		<ContextMenuItem onClick={handleRename}>
			<TextCursorInputIcon />
			Rename
		</ContextMenuItem>
	)
}

// ==================================

// tags: upload, uploadfile, uploadfilebutton, fileupload, fileuploadbutton
function UploadFileButton() {
	const { directory } = useContext(DirectoryPageContext)
	const setActiveDialogData = useSetAtom(activeDialogDataAtom)
	const inProgressFileUploadCount = useAtomValue(
		inProgressFileUploadCountAtom,
	)

	const handleClick = () => {
		setActiveDialogData({
			kind: DialogKind.UploadFile,
			directory: directory,
		})
	}

	if (inProgressFileUploadCount > 0) {
		return (
			<Button size="sm" type="button" loading onClick={handleClick}>
				Uploading {inProgressFileUploadCount} files
			</Button>
		)
	}

	return (
		<Button size="sm" type="button" onClick={handleClick}>
			Upload files
		</Button>
	)
}

function NewDirectoryItemDropdown() {
	const [activeDialogData, setActiveDialogData] =
		useAtom(activeDialogDataAtom)
	const store = useStore()

	const addNewDirectory = () => {
		setActiveDialogData({
			kind: DialogKind.NewDirectory,
		})
	}

	const handleCloseAutoFocus = (event: Event) => {
		// If we just created a new item, prevent the dropdown from restoring focus to the trigger
		if (activeDialogData) {
			event.preventDefault()
		}
	}

	return (
		<DropdownMenu>
			<DropdownMenuTrigger asChild>
				<Button size="sm" type="button" variant="outline">
					<PlusIcon className="size-4" />
					New
					<ChevronDownIcon className="pl-1 size-4 shrink-0" />
				</Button>
			</DropdownMenuTrigger>
			<DropdownMenuContent onCloseAutoFocus={handleCloseAutoFocus}>
				<DropdownMenuItem>
					<TextFileIcon />
					Text file
				</DropdownMenuItem>
				<DropdownMenuItem onClick={addNewDirectory}>
					<DirectoryIcon />
					Directory
				</DropdownMenuItem>
			</DropdownMenuContent>
		</DropdownMenu>
	)
}
