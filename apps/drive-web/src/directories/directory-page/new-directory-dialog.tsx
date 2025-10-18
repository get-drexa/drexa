import { api } from "@fileone/convex/api"
import type { Id } from "@fileone/convex/dataModel"
import { useMutation } from "@tanstack/react-query"
import { useMutation as useContextMutation } from "convex/react"
import { useId } from "react"
import { toast } from "sonner"
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

export function NewDirectoryDialog({
	open,
	onOpenChange,
	directoryId,
}: {
	open: boolean
	onOpenChange: (open: boolean) => void
	directoryId: Id<"directories">
}) {
	const formId = useId()

	const { mutate: createDirectory, isPending: isCreating } = useMutation({
		mutationFn: useContextMutation(api.files.createDirectory),
		onSuccess: () => {
			onOpenChange(false)
			toast.success("Directory created successfully")
		},
	})

	const onSubmit = (event: React.FormEvent<HTMLFormElement>) => {
		event.preventDefault()

		const formData = new FormData(event.currentTarget)
		const name = formData.get("directoryName") as string

		if (name) {
			createDirectory({ name, directoryId })
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
