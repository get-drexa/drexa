import { api } from "@fileone/convex/_generated/api"
import { type FileSystemItem, FileType } from "@fileone/convex/filesystem"
import { useMutation } from "@tanstack/react-query"
import { useMutation as useContextMutation } from "convex/react"
import { useId } from "react"
import { Button } from "@/components/ui/button"
import {
	Dialog,
	DialogClose,
	DialogContent,
	DialogFooter,
	DialogHeader,
	DialogTitle,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"

type RenameFileDialogProps = {
	item: FileSystemItem
	onRenameSuccess: () => void
	onClose: () => void
}

export function RenameFileDialog({
	item,
	onRenameSuccess,
	onClose,
}: RenameFileDialogProps) {
	const formId = useId()

	const { mutate: renameFile, isPending: isRenaming } = useMutation({
		mutationFn: useContextMutation(api.files.renameFile),
		onSuccess: () => {
			onRenameSuccess()
		},
	})

	const onSubmit = (event: React.FormEvent<HTMLFormElement>) => {
		event.preventDefault()

		const formData = new FormData(event.currentTarget)
		const newName = formData.get("itemName") as string

		if (newName) {
			switch (item.kind) {
				case FileType.File:
					renameFile({
						directoryId: item.doc.directoryId,
						itemId: item.doc._id,
						newName,
					})
					break
				default:
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
					<RenameFileInput initialValue={item.doc.name} />
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
