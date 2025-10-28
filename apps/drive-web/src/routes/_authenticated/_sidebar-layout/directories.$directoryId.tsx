import { api } from "@fileone/convex/api"
import type { Doc, Id } from "@fileone/convex/dataModel"
import {
	type FileSystemItem,
	newFileSystemHandle,
	type OpenedFile,
} from "@fileone/convex/filesystem"
import { useMutation } from "@tanstack/react-query"
import { createFileRoute } from "@tanstack/react-router"
import type { Row, Table } from "@tanstack/react-table"
import {
	useMutation as useContextMutation,
	useMutation as useConvexMutation,
	useQuery as useConvexQuery,
} from "convex/react"
import { atom, useAtom, useAtomValue, useSetAtom, useStore } from "jotai"
import {
	ChevronDownIcon,
	PlusIcon,
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
import { inProgressFileUploadCountAtom } from "@/files/store"
import { UploadFileDialog } from "@/files/upload-file-dialog"
import type { FileDragInfo } from "@/files/use-file-drop"

export const Route = createFileRoute(
	"/_authenticated/_sidebar-layout/directories/$directoryId",
)({
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
	directory: Doc<"directories">
}

type ActiveDialogData = NewDirectoryDialogData | UploadFileDialogData

// MARK: atoms
const contextMenuTargetItemsAtom = atom<FileSystemItem[]>([])
const activeDialogDataAtom = atom<ActiveDialogData | null>(null)
const fileDragInfoAtom = atom<FileDragInfo | null>(null)
const optimisticDeletedItemsAtom = atom(
	new Set<Id<"files"> | Id<"directories">>(),
)
const openedFileAtom = atom<OpenedFile | null>(null)
const itemBeingRenamedAtom = atom<{
	originalItem: FileSystemItem
	name: string
} | null>(null)

// MARK: page entry
function RouteComponent() {
	const { directoryId } = Route.useParams()
	const rootDirectory = useConvexQuery(api.files.fetchRootDirectory)
	const directory = useConvexQuery(api.files.fetchDirectory, {
		directoryId,
	})
	const directoryContent = useConvexQuery(
		api.filesystem.fetchDirectoryContent,
		{
			directoryId,
			trashed: false,
		},
	)

	const directoryUrlById = useCallback(
		(directoryId: Id<"directories">) => `/directories/${directoryId}`,
		[],
	)

	if (!directory || !directoryContent || !rootDirectory) {
		return <DirectoryPageSkeleton />
	}

	return (
		<DirectoryPageContext
			value={{ rootDirectory, directory, directoryContent }}
		>
			<header className="flex py-2 shrink-0 items-center gap-2 border-b px-4 w-full">
				<DirectoryPathBreadcrumb
					directory={directory}
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
					<_DirectoryContentTable />
				</div>
			</DirectoryContentContextMenu>

			<WithAtom atom={activeDialogDataAtom}>
				{(data, setData) => (
					<>
						<NewDirectoryDialog
							open={data?.kind === DialogKind.NewDirectory}
							directoryId={directory._id}
							onOpenChange={(open) => {
								if (!open) {
									setData(null)
								}
							}}
						/>
						{data?.kind === DialogKind.UploadFile && (
							<UploadFileDialog
								targetDirectory={data.directory}
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

// MARK: directory table

function _DirectoryContentTable() {
	const optimisticDeletedItems = useAtomValue(optimisticDeletedItemsAtom)
	const setOpenedFile = useSetAtom(openedFileAtom)
	const setContextMenuTargetItems = useSetAtom(contextMenuTargetItemsAtom)

	const { mutate: openFile } = useMutation({
		mutationFn: useConvexMutation(api.filesystem.openFile),
		onSuccess: (openedFile: OpenedFile) => {
			setOpenedFile(openedFile)
		},
		onError: (error) => {
			console.error(error)
			toast.error("Failed to open file")
		},
	})

	const onTableOpenFile = (file: Doc<"files">) => {
		openFile({ fileId: file._id })
	}

	const directoryUrlFn = useCallback(
		(directory: Doc<"directories">) => `/directories/${directory._id}`,
		[],
	)

	const handleContextMenuRequest = (
		row: Row<FileSystemItem>,
		table: Table<FileSystemItem>,
	) => {
		if (row.getIsSelected()) {
			setContextMenuTargetItems(
				table.getSelectedRowModel().rows.map((row) => row.original),
			)
		} else {
			setContextMenuTargetItems([row.original])
		}
	}

	return (
		<DirectoryContentTable
			hiddenItems={optimisticDeletedItems}
			directoryUrlFn={directoryUrlFn}
			fileDragInfoAtom={fileDragInfoAtom}
			onContextMenu={handleContextMenuRequest}
			onOpenFile={onTableOpenFile}
		/>
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
	const setOptimisticDeletedItems = useSetAtom(optimisticDeletedItemsAtom)
	const setBackgroundTaskProgress = useSetAtom(backgroundTaskProgressAtom)
	const moveToTrashMutation = useContextMutation(api.filesystem.moveToTrash)

	const { mutate: moveToTrash } = useMutation({
		mutationFn: moveToTrashMutation,
		onMutate: ({ handles }) => {
			setBackgroundTaskProgress({
				label: "Moving items to trash…",
			})
			setOptimisticDeletedItems(
				(prev) =>
					new Set([...prev, ...handles.map((handle) => handle.id)]),
			)
		},
		onSuccess: ({ deleted, errors }, { handles }) => {
			setBackgroundTaskProgress(null)
			setOptimisticDeletedItems((prev) => {
				const newSet = new Set(prev)
				for (const handle of handles) {
					newSet.delete(handle.id)
				}
				return newSet
			})
			if (errors.length === 0 && deleted.length === handles.length) {
				toast.success(`Moved ${handles.length} items to trash`)
			} else if (errors.length === handles.length) {
				toast.error("Failed to move to trash")
			} else {
				toast.info(
					`Moved ${deleted.length} items to trash; failed to move ${errors.length} items`,
				)
			}
		},
		onError: (_err, { handles }) => {
			setOptimisticDeletedItems((prev) => {
				const newSet = new Set(prev)
				for (const handle of handles) {
					newSet.delete(handle.id)
				}
				return newSet
			})
		},
	})

	const handleDelete = () => {
		const selectedItems = store.get(contextMenuTargetItemsAtom)
		if (selectedItems.length > 0) {
			moveToTrash({
				handles: selectedItems.map(newFileSystemHandle),
			})
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
				name: selectedItem.doc.name,
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
