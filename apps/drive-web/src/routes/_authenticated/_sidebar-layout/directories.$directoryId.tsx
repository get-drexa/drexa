import { useMutation, useQuery } from "@tanstack/react-query"
import { createFileRoute } from "@tanstack/react-router"
import type { Row, Table } from "@tanstack/react-table"
import { type } from "arktype"
import { atom, useAtom, useAtomValue, useSetAtom, useStore } from "jotai"
import {
	ChevronDownIcon,
	PlusIcon,
	ScissorsIcon,
	TextCursorInputIcon,
	TrashIcon,
} from "lucide-react"
import { useCallback, useContext } from "react"
import { toast } from "sonner"
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
import {
	DIRECTORY_CONTENT_ORDER_BY,
	DIRECTORY_CONTENT_ORDER_DIRECTION,
	directoryInfoQueryAtom,
	moveToTrashMutationAtom,
} from "@/vfs/api"
import type {
	DirectoryInfo,
	DirectoryInfoWithPath,
	DirectoryItem,
	FileInfo,
} from "@/vfs/vfs"

const DirectoryContentPageParams = type({
	orderBy: type
		.valueOf(DIRECTORY_CONTENT_ORDER_BY)
		.default(DIRECTORY_CONTENT_ORDER_BY.name),
	direction: type
		.valueOf(DIRECTORY_CONTENT_ORDER_DIRECTION)
		.default(DIRECTORY_CONTENT_ORDER_DIRECTION.asc),
})

export const Route = createFileRoute(
	"/_authenticated/_sidebar-layout/directories/$directoryId",
)({
	validateSearch: DirectoryContentPageParams,
	component: RouteComponent,
})

enum DialogKind {
	NewDirectory = "NewDirectory",
	UploadFile = "UploadFile",
}

type NewDirectoryDialogData = {
	kind: DialogKind.NewDirectory
}

type UploadFileDialogData = {
	kind: DialogKind.UploadFile
	directory: DirectoryInfoWithPath
}

type ActiveDialogData = NewDirectoryDialogData | UploadFileDialogData

// MARK: atoms
const contextMenuTargetItemsAtom = atom<DirectoryItem[]>([])
const activeDialogDataAtom = atom<ActiveDialogData | null>(null)
const fileDragInfoAtom = atom<FileDragInfo | null>(null)
const optimisticDeletedItemsAtom = atom(new Set<string>())
const openedFileAtom = atom<FileInfo | null>(null)
const itemBeingRenamedAtom = atom<{
	originalItem: DirectoryItem
	name: string
} | null>(null)

// MARK: page entry
function RouteComponent() {
	const { directoryId } = Route.useParams()
	const { data: directoryInfo, isLoading: isLoadingDirectoryInfo } = useQuery(
		useAtomValue(directoryInfoQueryAtom(directoryId)),
	)

	const setOpenedFile = useSetAtom(openedFileAtom)
	const setContextMenuTargetItems = useSetAtom(contextMenuTargetItemsAtom)

	const directoryUrlById = useCallback(
		(directoryId: string) => `/directories/${directoryId}`,
		[],
	)

	const onTableOpenFile = useCallback(
		(file: FileInfo) => {
			setOpenedFile(file)
		},
		[setOpenedFile],
	)

	const directoryUrlFn = useCallback(
		(directory: DirectoryInfo) => `/directories/${directory.id}`,
		[],
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
					<NewDirectoryItemDropdown />
					<UploadFileButton />
				</div>
			</header>

			{/* DirectoryContentContextMenu must wrap div instead of DirectoryContentTable, otherwise radix will throw "event.preventDefault is not a function" error, idk why */}
			<DirectoryContentContextMenu>
				<div className="w-full">
					<DirectoryContentTable
						directoryUrlFn={directoryUrlFn}
						fileDragInfoAtom={fileDragInfoAtom}
						onContextMenu={handleContextMenuRequest}
						onOpenFile={onTableOpenFile}
					/>
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

	const moveToTrashMutation = useAtomValue(moveToTrashMutationAtom)
	const { mutate: moveToTrash } = useMutation({
		...moveToTrashMutation,
		onMutate: (vars, ctx) => {
			setBackgroundTaskProgress({
				label: "Moving items to trash…",
			})
			return (
				moveToTrashMutation.onMutate?.(vars, ctx) ?? {
					prevDirContentMap: new Map(),
				}
			)
		},
		onSuccess: (data, vars, result, ctx) => {
			moveToTrashMutation.onSuccess?.(data, vars, result, ctx)
			setBackgroundTaskProgress(null)
			toast.success(`Moved ${data.length} items to trash`)
		},
		onError: (err, vars, mutateResult, context) => {
			moveToTrashMutation.onError?.(err, vars, mutateResult, context)
			toast.error(formatError(err))
			setBackgroundTaskProgress(null)
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
