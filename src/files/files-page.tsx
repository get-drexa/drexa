import { api } from "@convex/_generated/api"
import { useMutation } from "@tanstack/react-query"
import { useMutation as useConvexMutation } from "convex/react"
import { useSetAtom } from "jotai"
import {
	ChevronDownIcon,
	Loader2Icon,
	PlusIcon,
	UploadCloudIcon,
} from "lucide-react"
import { type ChangeEvent, useRef } from "react"
import { toast } from "sonner"
import {
	DropdownMenu,
	DropdownMenuContent,
	DropdownMenuItem,
	DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { DirectoryIcon } from "../components/icons/directory-icon"
import { TextFileIcon } from "../components/icons/text-file-icon"
import {
	Breadcrumb,
	BreadcrumbItem,
	BreadcrumbList,
	BreadcrumbPage,
} from "../components/ui/breadcrumb"
import { Button } from "../components/ui/button"
import { SidebarTrigger } from "../components/ui/sidebar"
import { FileTable } from "./file-table"
import { newItemKindAtom } from "./state"

export function FilesPage() {
	return (
		<>
			<header className="flex py-2 shrink-0 items-center gap-2 border-b px-4 w-full">
				<SidebarTrigger className="-ml-1" />
				<Breadcrumb>
					<BreadcrumbList>
						<BreadcrumbItem>
							<BreadcrumbPage>Files</BreadcrumbPage>
						</BreadcrumbItem>
					</BreadcrumbList>
				</Breadcrumb>
				<div className="ml-auto flex flex-row gap-2">
					<NewDirectoryItemDropdown />
					<UploadFileButton />
				</div>
			</header>
			<div className="p-4 w-full">
				<FileTable />
			</div>
		</>
	)
}

// tags: upload, uploadfile, uploadfilebutton, fileupload, fileuploadbutton
function UploadFileButton() {
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
	const setNewItemKind = useSetAtom(newItemKindAtom)

	const addNewDirectory = () => {
		setNewItemKind("directory")
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
			<DropdownMenuContent>
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
