import { api } from "@fileone/convex/api"
import { useMutation } from "@tanstack/react-query"
import { useMutation as useContextMutation } from "convex/react"
import { atom, useAtom, useStore } from "jotai"
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
import { itemBeingRenamedAtom } from "./state"

const fielNameAtom = atom(
	(get) => get(itemBeingRenamedAtom)?.name,
	(get, set, newName: string) => {
		const current = get(itemBeingRenamedAtom)
		if (current) {
			set(itemBeingRenamedAtom, {
				...current,
				name: newName,
			})
		}
	},
)

export function RenameFileDialog() {
	const [itemBeingRenamed, setItemBeingRenamed] =
		useAtom(itemBeingRenamedAtom)
	const store = useStore()
	const formId = useId()

	const { mutate: renameFile, isPending: isRenaming } = useMutation({
		mutationFn: useContextMutation(api.files.renameFile),
		onSuccess: () => {
			setItemBeingRenamed(null)
			toast.success("File renamed successfully")
		},
	})

	const onSubmit = (event: React.FormEvent<HTMLFormElement>) => {
		event.preventDefault()

		const itemBeingRenamed = store.get(itemBeingRenamedAtom)
		if (itemBeingRenamed) {
			const formData = new FormData(event.currentTarget)
			const newName = formData.get("itemName") as string

			if (newName) {
				switch (itemBeingRenamed.originalItem.kind) {
					case "file":
						renameFile({
							directoryId:
								itemBeingRenamed.originalItem.doc.directoryId,
							itemId: itemBeingRenamed.originalItem.doc._id,
							newName,
						})
						break
					default:
						break
				}
			}
		}
	}

	return (
		<Dialog
			open={itemBeingRenamed !== null}
			onOpenChange={(open) =>
				setItemBeingRenamed(open ? itemBeingRenamed : null)
			}
		>
			<DialogContent className="sm:max-w-md">
				<DialogHeader>
					<DialogTitle>Rename File</DialogTitle>
				</DialogHeader>

				<form id={formId} onSubmit={onSubmit}>
					<RenameFileInput />
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

function RenameFileInput() {
	const [fileName, setFileName] = useAtom(fielNameAtom)
	return (
		<Input
			value={fileName}
			name="itemName"
			onChange={(e) => setFileName(e.target.value)}
		/>
	)
}
