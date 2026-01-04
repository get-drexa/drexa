import { useMutation } from "@tanstack/react-query"
import { useAtomValue } from "jotai"
import { useId } from "react"
import { toast } from "sonner"
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
import { createDirectoryMutationOptions } from "@/vfs/api"
import type { DirectoryInfo } from "@/vfs/vfs"

export function NewDirectoryDialog({
	open,
	onOpenChange,
	parentDirectory,
}: {
	open: boolean
	onOpenChange: (open: boolean) => void
	parentDirectory: DirectoryInfo
}) {
	const formId = useId()

	const org = useCurrentOrganization()
	const drive = useAtomValue(currentDriveAtom)
	const createDirectoryMutation = createDirectoryMutationOptions({ org, drive })

	const { mutate: createDirectory, isPending: isCreating } = useMutation({
		...createDirectoryMutation,
		onSuccess: (data, vars, result, context) => {
			createDirectoryMutation.onSuccess?.(data, vars, result, context)
			onOpenChange(false)
			toast.success("Directory created successfully")
		},
	})

	const onSubmit = (event: React.FormEvent<HTMLFormElement>) => {
		event.preventDefault()

		const formData = new FormData(event.currentTarget)
		const name = formData.get("directoryName") as string

		if (name) {
			createDirectory({ name, parentId: parentDirectory.id })
		}
	}

	return (
		<Dialog open={open} onOpenChange={onOpenChange}>
			<DialogContent className="sm:max-w-md">
				<DialogHeader>
					<DialogTitle>New Directory</DialogTitle>
				</DialogHeader>

				<form id={formId} onSubmit={onSubmit}>
					<Input name="directoryName" />
				</form>

				<DialogFooter>
					<DialogClose asChild>
						<Button loading={isCreating} variant="outline">
							<span>Cancel</span>
						</Button>
					</DialogClose>
					<Button loading={isCreating} type="submit" form={formId}>
						<span>Create</span>
					</Button>
				</DialogFooter>
			</DialogContent>
		</Dialog>
	)
}
