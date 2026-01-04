import { useMutation } from "@tanstack/react-query"
import { useAtomValue } from "jotai"
import { useId } from "react"
import { Button } from "@/components/ui/button"
import { currentDriveAtom } from "@/drive/drive"
import { useCurrentOrganization } from "@/organization/context"
import {
	Dialog,
	DialogClose,
	DialogContent,
	DialogFooter,
	DialogHeader,
	DialogTitle,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import {
	renameDirectoryMutationOptions,
	renameFileMutationOptions,
} from "@/vfs/api"
import type { DirectoryItem } from "@/vfs/vfs"

type RenameFileDialogProps = {
	item: DirectoryItem
	onRenameSuccess: () => void
	onClose: () => void
}

export function RenameFileDialog({
	item,
	onRenameSuccess,
	onClose,
}: RenameFileDialogProps) {
	const formId = useId()

	const org = useCurrentOrganization()
	const drive = useAtomValue(currentDriveAtom)
	const renameFileMutation = renameFileMutationOptions({ org, drive })
	const renameDirectoryMutation = renameDirectoryMutationOptions({ org, drive })

	const { mutate: renameFile, isPending: isRenamingFile } = useMutation({
		...renameFileMutation,
		onSuccess: onRenameSuccess,
	})

	const { mutate: renameDirectory, isPending: isRenamingDirectory } =
		useMutation({
			...renameDirectoryMutation,
			onSuccess: onRenameSuccess,
		})

	const isRenaming = isRenamingFile || isRenamingDirectory

	const onSubmit = (event: React.FormEvent<HTMLFormElement>) => {
		event.preventDefault()

		const formData = new FormData(event.currentTarget)
		const newName = formData.get("itemName") as string

		if (newName) {
			switch (item.kind) {
				case "file":
					renameFile(item)
					break
				case "directory":
					renameDirectory(item)
					break
			}
		}
	}

	return (
		<Dialog
			open
			onOpenChange={(open) => {
				if (!open) {
					onClose()
				}
			}}
		>
			<DialogContent className="sm:max-w-md">
				<DialogHeader>
					<DialogTitle>Rename File</DialogTitle>
				</DialogHeader>

				<form id={formId} onSubmit={onSubmit}>
					<RenameFileInput initialValue={item.name} />
				</form>

				<DialogFooter>
					<DialogClose asChild>
						<Button loading={isRenaming} variant="outline">
							<span>Cancel</span>
						</Button>
					</DialogClose>
					<Button loading={isRenaming} type="submit" form={formId}>
						<span>Rename</span>
					</Button>
				</DialogFooter>
			</DialogContent>
		</Dialog>
	)
}

function RenameFileInput({ initialValue }: { initialValue: string }) {
	return <Input defaultValue={initialValue} name="itemName" />
}
