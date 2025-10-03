import { api } from "@fileone/convex/_generated/api"
import { FileType } from "@fileone/convex/model/filesystem"
import { useMutation } from "@tanstack/react-query"
import { createFileRoute } from "@tanstack/react-router"
import {
	useMutation as useConvexMutation,
	useQuery as useConvexQuery,
} from "convex/react"
import { atom, useAtom } from "jotai"
import {
	ChevronDownIcon,
	Loader2Icon,
	PlusIcon,
	UploadCloudIcon,
} from "lucide-react"
import { type ChangeEvent, useContext, useRef } from "react"
import { toast } from "sonner"
import { DirectoryIcon } from "@/components/icons/directory-icon"
import { TextFileIcon } from "@/components/icons/text-file-icon"
import { Button } from "@/components/ui/button"
import {
	DropdownMenu,
	DropdownMenuContent,
	DropdownMenuItem,
	DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { WithAtom } from "@/components/with-atom"
import { DirectoryPageContext } from "@/directories/directory-page/context"
import { DirectoryPage } from "@/directories/directory-page/directory-page"
import { DirectoryPageSkeleton } from "@/directories/directory-page/directory-page-skeleton"
import { FilePathBreadcrumb } from "@/directories/directory-page/file-path-breadcrumb"
import { NewDirectoryDialog } from "@/directories/directory-page/new-directory-dialog"
import { RenameFileDialog } from "@/directories/directory-page/rename-file-dialog"

export const Route = createFileRoute(
	"/_authenticated/_sidebar-layout/directories/$directoryId",
)({
	component: RouteComponent,
})

const newFileTypeAtom = atom<FileType | null>(null)

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

	if (!directory || !directoryContent || !rootDirectory) {
		return <DirectoryPageSkeleton />
	}

	return (
		<DirectoryPageContext
			value={{ rootDirectory, directory, directoryContent }}
		>
			<header className="flex py-2 shrink-0 items-center gap-2 border-b px-4 w-full">
				<FilePathBreadcrumb />
				<div className="ml-auto flex flex-row gap-2">
					<NewDirectoryItemDropdown />
					<UploadFileButton />
				</div>
			</header>

			<DirectoryPage />

			<WithAtom atom={newFileTypeAtom}>
				{(newFileType, setNewFileType) => (
					<NewDirectoryDialog
						open={newFileType === FileType.Directory}
						directoryId={directory._id}
						onOpenChange={(open) => {
							if (!open) {
								setNewFileType(null)
							}
						}}
					/>
				)}
			</WithAtom>

			<RenameFileDialog />
		</DirectoryPageContext>
	)
}

// tags: upload, uploadfile, uploadfilebutton, fileupload, fileuploadbutton
function UploadFileButton() {
	const { directory } = useContext(DirectoryPageContext)
	const generateUploadUrl = useConvexMutation(api.files.generateUploadUrl)
	const saveFile = useConvexMutation(api.files.saveFile)
	const { mutate: uploadFile, isPending: isUploading } = useMutation({
		mutationFn: async (file: File) => {
			const uploadUrl = await generateUploadUrl()
			const uploadResult = await fetch(uploadUrl, {
				method: "POST",
				body: file,
				headers: {
					"Content-Type": file.type,
				},
			})
			const { storageId } = await uploadResult.json()

			await saveFile({
				storageId,
				name: file.name,
				size: file.size,
				mimeType: file.type,
				directoryId: directory._id,
			})
		},
		onSuccess: () => {
			toast.success("File uploaded successfully.")
		},
	})

	const fileInputRef = useRef<HTMLInputElement>(null)

	const handleClick = () => {
		fileInputRef.current?.click()
	}

	const onFileUpload = async (e: ChangeEvent<HTMLInputElement>) => {
		const file = e.target.files?.[0]
		if (file) {
			uploadFile(file)
		}
	}

	return (
		<>
			<input
				hidden
				onChange={onFileUpload}
				ref={fileInputRef}
				type="file"
				name="files"
			/>
			<Button
				size="sm"
				type="button"
				onClick={handleClick}
				disabled={isUploading}
			>
				{isUploading ? (
					<Loader2Icon className="animate-spin size-4" />
				) : (
					<UploadCloudIcon className="size-4" />
				)}
				{isUploading ? "Uploading" : "Upload File"}
			</Button>
		</>
	)
}

function NewDirectoryItemDropdown() {
	const [newFileType, setNewFileType] = useAtom(newFileTypeAtom)

	const addNewDirectory = () => {
		setNewFileType(FileType.Directory)
	}

	const handleCloseAutoFocus = (event: Event) => {
		// If we just created a new item, prevent the dropdown from restoring focus to the trigger
		if (newFileType) {
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
