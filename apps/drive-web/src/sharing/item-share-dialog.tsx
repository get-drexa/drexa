import { useMutation, useQuery } from "@tanstack/react-query"
import { useAtomValue, useStore } from "jotai"
import {
	CheckIcon,
	CopyIcon,
	EllipsisIcon,
	LinkIcon,
	LockKeyholeIcon,
} from "lucide-react"
import { createContext, useContext, useRef } from "react"
import {
	CrossfadeIcon,
	type CrossfadeIconHandle,
} from "@/components/crossfade-icon"
import { Button } from "@/components/ui/button"
import { ButtonGroup } from "@/components/ui/button-group"
import {
	Dialog,
	DialogContent,
	DialogDescription,
	DialogHeader,
	DialogTitle,
} from "@/components/ui/dialog"
import {
	DropdownMenu,
	DropdownMenuContent,
	DropdownMenuGroup,
	DropdownMenuItem,
	DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { copyToClipboardMutation } from "@/lib/clipboard"
import {
	createShareMutationAtom,
	deleteShareMutationAtom,
	directorySharesQueryAtom,
	fileSharesQueryAtom,
} from "@/sharing/api"
import type { DirectoryItem } from "@/vfs/vfs"
import {
	Tooltip,
	TooltipContent,
	TooltipTrigger,
} from "../components/ui/tooltip"
import type { Share } from "./share"

type ItemShareDialogProps = {
	item: DirectoryItem | null
	open: boolean
	onClose: () => void
}

const ItemShareDialogContext = createContext<{
	item: DirectoryItem
}>(
	null as unknown as {
		item: DirectoryItem
	},
)

export function ItemShareDialog({ item, open, onClose }: ItemShareDialogProps) {
	let description: string
	switch (item?.kind) {
		case "file":
			description = "Configure external access to this file."
			break
		case "directory":
			description = "Configure external access to this directory."
			break
		default:
			description = "Configure external access to this item."
			break
	}

	return (
		<Dialog open={open} onOpenChange={onClose}>
			<DialogContent>
				<DialogHeader>
					<DialogTitle>Share {item?.name}</DialogTitle>
					<DialogDescription>{description}</DialogDescription>
				</DialogHeader>
				{item && (
					<ItemShareDialogContext value={{ item }}>
						<div>
							<PublicAccessSection item={item} />
						</div>
					</ItemShareDialogContext>
				)}
			</DialogContent>
		</Dialog>
	)
}

function PublicAccessSection({ item }: { item: DirectoryItem }) {
	const fileSharesQuery = useAtomValue(fileSharesQueryAtom(item.id))
	const directorySharesQuery = useAtomValue(directorySharesQueryAtom(item.id))

	const { data: fileShares, isLoading: isLoadingFileShares } = useQuery({
		...fileSharesQuery,
		enabled: item.kind === "file",
	})
	const { data: directoryShares, isLoading: isLoadingDirectoryShares } =
		useQuery({
			...directorySharesQuery,
			enabled: item.kind === "directory",
		})

	let shares: Share[] = []
	if (fileShares) {
		shares = fileShares
	} else if (directoryShares) {
		shares = directoryShares
	}

	let content: React.ReactNode = null
	if (isLoadingFileShares || isLoadingDirectoryShares) {
		content = <div>Loading...</div>
	} else if (shares.length === 0) {
		content = (
			<div className="flex items-center gap-3">
				<div className="bg-muted rounded-md p-1.5 translate-y-px border border-border shadow-xs">
					<LockKeyholeIcon size={16} />
				</div>
				<div className="flex grow flex-col text-sm">
					<p className="font-medium">No share link created</p>
					<p className="text-muted-foreground">
						Only you can access this item.
					</p>
				</div>
				<CreateShareLinkButton />
			</div>
		)
	} else {
		content = (
			<ul>
				{shares.map((share) => (
					<ShareLinkListItem key={share.id} share={share} />
				))}
			</ul>
		)
	}

	return (
		<div>
			<p className="text-md mb-2">Public Access</p>
			{content}
		</div>
	)
}

function ShareLinkListItem({ share }: { share: Share }) {
	const { item } = useContext(ItemShareDialogContext)
	const copyLinkButtonRef = useRef<HTMLButtonElement>(null)
	const copyIconRef = useRef<CrossfadeIconHandle>(null)

	const { mutate: copyToClipboard } = useMutation({
		...copyToClipboardMutation,
		onSuccess: () => {
			copyIconRef.current?.trigger()
		},
	})

	const copyItemShareLinkToClipboard = () => {
		let link: string
		switch (item.kind) {
			case "file":
				link = `${window.location.origin}/shares/${share.id}/files/${item.id}`
				break
			case "directory":
				link = `${window.location.origin}/shares/${share.id}/directories/${item.id}`
				break
			default:
				link = ""
				break
		}
		if (link) {
			copyToClipboard(link)
		}
	}

	return (
		<li key={share.id} className="group flex items-center gap-3">
			{/** biome-ignore lint/a11y/noStaticElementInteractions: this is strictly for convenience. the normal copy link button is still accessible. */}
			{/** biome-ignore lint/a11y/useKeyWithClickEvents: this is strictly for convenience. the normal copy link button is still accessible. */}
			<div
				className="bg-muted rounded-md p-1.5 translate-y-px border border-border shadow-xs"
				onClick={() => {
					copyLinkButtonRef.current?.click()
				}}
			>
				<LinkIcon size={16} />
			</div>
			{/** biome-ignore lint/a11y/noStaticElementInteractions: this is strictly for convenience. the normal copy link button is still accessible. */}
			{/** biome-ignore lint/a11y/useKeyWithClickEvents: this is strictly for convenience. the normal copy link button is still accessible. */}
			<div
				className="flex grow flex-col text-sm"
				onClick={() => {
					copyLinkButtonRef.current?.click()
				}}
			>
				<p>Share link</p>
				<p className="text-muted-foreground">
					{share.expiresAt
						? `Expires at ${share.expiresAt}`
						: "Never expires"}
				</p>
			</div>
			<ButtonGroup>
				<Tooltip defaultOpen={false} delayDuration={1000}>
					<TooltipTrigger asChild>
						<Button
							tabIndex={-1}
							ref={copyLinkButtonRef}
							variant="outline"
							size="sm"
							className="group-[&:hover:not(:has(.share-options-trigger:hover))]:bg-accent group-[&:hover:not(:has(.share-options-trigger:hover))]:text-accent-foreground dark:group-[&:hover:not(:has(.share-options-trigger:hover))]:bg-input/50"
							onClick={copyItemShareLinkToClipboard}
						>
							<CrossfadeIcon
								ref={copyIconRef}
								from={<CopyIcon />}
								to={<CheckIcon />}
							/>
							<span className="sr-only">Copy share link</span>
						</Button>
					</TooltipTrigger>
					<TooltipContent>Copy share link</TooltipContent>
				</Tooltip>
				<DropdownMenu>
					<DropdownMenuTrigger asChild>
						<Button
							variant="outline"
							size="sm"
							className="share-options-trigger aria-expanded:bg-accent!"
						>
							<EllipsisIcon />
							<span className="sr-only">Share link options</span>
						</Button>
					</DropdownMenuTrigger>
					<ShareLinkOptionsMenu share={share} />
				</DropdownMenu>
			</ButtonGroup>
		</li>
	)
}

function ShareLinkOptionsMenu({ share }: { share: Share }) {
	const { item } = useContext(ItemShareDialogContext)
	const store = useStore()

	const { mutate: deleteShare } = useMutation({
		...useAtomValue(deleteShareMutationAtom),
		onMutate: ({ shareId }, { client }) => {
			let queryKey: readonly unknown[] | null
			switch (item.kind) {
				case "file":
					queryKey = store.get(fileSharesQueryAtom(item.id)).queryKey
					break
				case "directory":
					queryKey = store.get(
						directorySharesQueryAtom(item.id),
					).queryKey
					break
				default:
					queryKey = null
					break
			}
			if (queryKey) {
				const prevShares = client.getQueryData<Share[]>(queryKey)
				client.setQueryData<Share[]>(
					queryKey,
					(old) => old?.filter((s) => s.id !== shareId) ?? old,
				)
				return { queryKey, prevShares }
			}
			return null
		},
		onSuccess: (_data, _vars, mutateResult, { client }) => {
			if (mutateResult) {
				client.invalidateQueries({
					queryKey: mutateResult.queryKey,
				})
			}
		},
		onError: (error, _vars, mutateResult, { client }) => {
			console.error(error)
			if (mutateResult) {
				client.setQueryData<Share[]>(
					mutateResult.queryKey,
					mutateResult.prevShares,
				)
			}
		},
	})

	return (
		<DropdownMenuContent align="end">
			<DropdownMenuGroup>
				<DropdownMenuItem>Rename link</DropdownMenuItem>
				<DropdownMenuItem>Set expiration</DropdownMenuItem>
				<DropdownMenuItem
					variant="destructive"
					onClick={() => {
						deleteShare({ shareId: share.id })
					}}
				>
					Delete link
				</DropdownMenuItem>
			</DropdownMenuGroup>
		</DropdownMenuContent>
	)
}

function CreateShareLinkButton() {
	const { item } = useContext(ItemShareDialogContext)
	const store = useStore()
	const { mutate: createShare, isPending: isCreatingShare } = useMutation({
		...useAtomValue(createShareMutationAtom),
		onSuccess: (_createdShare, _vars, _, { client }) => {
			let queryKey: readonly unknown[] | null
			switch (item.kind) {
				case "file":
					queryKey = store.get(fileSharesQueryAtom(item.id)).queryKey
					break
				case "directory":
					queryKey = store.get(
						directorySharesQueryAtom(item.id),
					).queryKey
					break
				default:
					queryKey = null
					break
			}
			if (queryKey) {
				client.invalidateQueries({
					queryKey,
				})
			}
		},
	})

	return (
		<Button
			variant="outline"
			size="sm"
			loading={isCreatingShare}
			disabled={isCreatingShare}
			onClick={() => {
				createShare({ items: [item.id] })
			}}
		>
			Create share link
		</Button>
	)
}
