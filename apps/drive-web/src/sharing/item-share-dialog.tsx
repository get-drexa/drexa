import { useMutation, useQuery } from "@tanstack/react-query"
import {
	atom,
	type PrimitiveAtom,
	useAtomValue,
	useSetAtom,
	useStore,
} from "jotai"
import {
	CheckIcon,
	CopyIcon,
	EllipsisIcon,
	Link2OffIcon,
	LinkIcon,
	LockKeyholeIcon,
	PlusIcon,
} from "lucide-react"
import type React from "react"
import { createContext, useContext, useMemo, useRef, useState } from "react"
import {
	CrossfadeIcon,
	type CrossfadeIconHandle,
} from "@/components/crossfade-icon"
import { DateInput, type DateInputHandle } from "@/components/date-input"
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
import { Input } from "@/components/ui/input"
import { Kbd } from "@/components/ui/kbd"
import { LoadingSpinner } from "@/components/ui/loading-spinner"
import {
	Popover,
	PopoverContent,
	PopoverTrigger,
} from "@/components/ui/popover"
import {
	Select,
	SelectContent,
	SelectItem,
	SelectTrigger,
	SelectValue,
} from "@/components/ui/select"
import { Separator } from "@/components/ui/separator"
import {
	Tooltip,
	TooltipContent,
	TooltipTrigger,
} from "@/components/ui/tooltip"
import { copyToClipboardMutation } from "@/lib/clipboard"
import { cn } from "@/lib/utils"
import {
	createShareMutationAtom,
	deleteShareMutationAtom,
	directorySharesQueryAtom,
	fileSharesQueryAtom,
	updateShareMutationAtom,
} from "@/sharing/api"
import type { DirectoryItem } from "@/vfs/vfs"
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
			<DialogContent className="select-none">
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

	const sortShares = (shares: Share[]) =>
		[...shares].sort((a, b) => {
			if (a.expiresAt && b.expiresAt) {
				return a.expiresAt.getTime() - b.expiresAt.getTime()
			}
			return 0
		})

	const { data: fileShares, isLoading: isLoadingFileShares } = useQuery({
		...fileSharesQuery,
		enabled: item.kind === "file",
		select: sortShares,
	})
	const { data: directoryShares, isLoading: isLoadingDirectoryShares } =
		useQuery({
			...directorySharesQuery,
			enabled: item.kind === "directory",
			select: sortShares,
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
			<ul className="space-y-2">
				{shares.map((share) => (
					<ShareLinkListItem key={share.id} share={share} />
				))}
				<Separator className="my-2" />
				<AddShareLinkListItem />
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

	const isExpired = share.expiresAt && share.expiresAt.getTime() < Date.now()
	const formattedExpirationDate = share.expiresAt
		? share.expiresAt.toLocaleDateString("en-US", {
				month: "long",
				day: "numeric",
				year: "numeric",
			})
		: ""

	let statusMessage: string
	if (isExpired) {
		statusMessage = `Expired on ${formattedExpirationDate}`
	} else if (share.expiresAt) {
		statusMessage = `Expires on ${formattedExpirationDate}`
	} else {
		statusMessage = "Never expires"
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
				{isExpired ? (
					<Link2OffIcon size={16} />
				) : (
					<LinkIcon size={16} />
				)}
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
				<p
					className={cn(
						isExpired
							? "text-amber-700/80 dark:text-amber-400/80"
							: "text-muted-foreground",
					)}
				>
					{statusMessage}
				</p>
			</div>
			<ButtonGroup>
				{isExpired ? null : (
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
				)}
				<ShareLinkOptionsMenuButton share={share} />
			</ButtonGroup>
		</li>
	)
}

function AddShareLinkListItem() {
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
		<li>
			<button
				type="button"
				className="flex w-full items-center gap-3 text-sm font-regular group"
				onClick={() => {
					createShare({ items: [item.id] })
				}}
			>
				<span className="bg-secondary rounded-md p-1.5 translate-y-px border border-border shadow-xs group-hover:bg-secondary">
					{isCreatingShare ? (
						<LoadingSpinner />
					) : (
						<PlusIcon size={16} />
					)}
				</span>
				Create share link
			</button>
		</li>
	)
}

const ACTIVE_POPOVER_KIND = {
	rename: "rename",
	setExpiration: "setExpiration",
} as const
type ActivePopoverKind =
	(typeof ACTIVE_POPOVER_KIND)[keyof typeof ACTIVE_POPOVER_KIND]

function ShareLinkOptionsMenuButton({ share }: { share: Share }) {
	const activePopoverAtom = useMemo(
		() => atom<ActivePopoverKind | null>(null),
		[],
	)

	const activePopoverKind = useAtomValue(activePopoverAtom)

	const button = (
		<Button
			variant="outline"
			size="sm"
			className={cn("share-options-trigger aria-expanded:bg-accent!", {
				"bg-accent!": activePopoverKind !== null,
			})}
		>
			<EllipsisIcon />
			<span className="sr-only">Share link options</span>
		</Button>
	)

	switch (activePopoverKind) {
		case "rename":
			return (
				<RenameShareLinkPopover
					share={share}
					activePopoverAtom={activePopoverAtom}
				>
					{button}
				</RenameShareLinkPopover>
			)

		case "setExpiration":
			return (
				<ConfigureShareLinkExpirationPopover
					share={share}
					activePopoverAtom={activePopoverAtom}
				>
					{button}
				</ConfigureShareLinkExpirationPopover>
			)

		default:
			return (
				<ShareLinkOptionsMenu
					share={share}
					activePopoverAtom={activePopoverAtom}
				>
					{button}
				</ShareLinkOptionsMenu>
			)
	}
}

function RenameShareLinkPopover({
	share,
	children,
	activePopoverAtom,
}: React.PropsWithChildren<{
	share: Share
	activePopoverAtom: PrimitiveAtom<ActivePopoverKind | null>
}>) {
	const setActivePopover = useSetAtom(activePopoverAtom)
	const inputId = `rename-share-link-${share.id}`

	return (
		<Popover
			modal
			defaultOpen
			onOpenChange={(nextOpen) => {
				if (!nextOpen) {
					setActivePopover(null)
				}
			}}
		>
			<PopoverTrigger asChild>{children}</PopoverTrigger>
			<PopoverContent align="end">
				<header className="mb-3">
					<h1 className="leading-none font-medium mb-2">
						Set a label for this link
					</h1>
					<p className="text-muted-foreground text-sm">
						Only you can see the label.
					</p>
				</header>
				<Input
					id={inputId}
					name={inputId}
					placeholder="e.g. Product roadmap share"
				/>
				<div className="flex justify-end mt-4 gap-2">
					<Button
						variant="outline"
						size="sm"
						onClick={() => {
							setActivePopover(null)
						}}
					>
						Cancel
					</Button>
					<Button variant="default" size="sm">
						Save{" "}
						<Kbd className="border border-primary-foreground/20 bg-transparent text-primary-foreground">
							⏎
						</Kbd>
					</Button>
				</div>
			</PopoverContent>
		</Popover>
	)
}

function ConfigureShareLinkExpirationPopover({
	share,
	children,
	activePopoverAtom,
}: React.PropsWithChildren<{
	share: Share
	activePopoverAtom: PrimitiveAtom<ActivePopoverKind | null>
}>) {
	const EXPIRATION_TYPE = {
		date: "date",
		never: "never",
	} as const
	type ExpirationType = (typeof EXPIRATION_TYPE)[keyof typeof EXPIRATION_TYPE]

	const [expirationType, setExpirationType] = useState<ExpirationType>(
		share.expiresAt === null ? EXPIRATION_TYPE.never : EXPIRATION_TYPE.date,
	)
	const { item } = useContext(ItemShareDialogContext)
	const store = useStore()
	const setActivePopover = useSetAtom(activePopoverAtom)
	const dateInputRef = useRef<DateInputHandle>(null)

	const { mutate: updateShare, isPending: isUpdatingShare } = useMutation({
		...useAtomValue(updateShareMutationAtom),
		onSuccess: (_updatedShare, _vars, _, { client }) => {
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
			setActivePopover(null)
		},
	})

	const updateExpirationDate = () => {
		switch (expirationType) {
			case EXPIRATION_TYPE.date:
				if (dateInputRef.current?.date) {
					updateShare({
						shareId: share.id,
						expiresAt: dateInputRef.current.date,
					})
				}
				break
			case EXPIRATION_TYPE.never:
				updateShare({
					shareId: share.id,
					expiresAt: null,
				})
				break
		}
	}

	return (
		<Popover
			modal
			defaultOpen
			onOpenChange={(nextOpen) => {
				if (!nextOpen) {
					setActivePopover(null)
				}
			}}
		>
			<PopoverTrigger asChild>{children}</PopoverTrigger>
			<PopoverContent align="end">
				<header className="mb-2">
					<h1 className="leading-none font-medium mb-2">
						Configure expiration
					</h1>
					<p className="text-muted-foreground text-sm">
						The share link will be accessible until the selected
						date.
					</p>
				</header>
				<Select
					value={expirationType}
					onValueChange={(value) => {
						setExpirationType(value as ExpirationType)
					}}
				>
					<SelectTrigger className="w-full">
						<SelectValue placeholder="Select a fruit" />
					</SelectTrigger>
					<SelectContent>
						<SelectItem value="date">
							Link expires on a date
						</SelectItem>
						<SelectItem value="never">
							Link never expires
						</SelectItem>
					</SelectContent>
				</Select>
				{expirationType === EXPIRATION_TYPE.date ? (
					<DateInput
						className="mt-2"
						ref={dateInputRef}
						defaultValue={share.expiresAt ?? undefined}
					/>
				) : null}
				<div className="flex justify-end mt-4 gap-2">
					<Button
						variant="outline"
						size="sm"
						disabled={isUpdatingShare}
						onClick={() => {
							setActivePopover(null)
						}}
					>
						Cancel
					</Button>
					<Button
						variant="default"
						size="sm"
						loading={isUpdatingShare}
						disabled={isUpdatingShare}
						onClick={updateExpirationDate}
					>
						Save{" "}
						<Kbd className="border border-primary-foreground/20 bg-transparent text-primary-foreground">
							⏎
						</Kbd>
					</Button>
				</div>
			</PopoverContent>
		</Popover>
	)
}

function ShareLinkOptionsMenu({
	share,
	children,
	activePopoverAtom,
}: React.PropsWithChildren<{
	share: Share
	activePopoverAtom: PrimitiveAtom<ActivePopoverKind | null>
}>) {
	const { item } = useContext(ItemShareDialogContext)
	const setActivePopover = useSetAtom(activePopoverAtom)
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
		<DropdownMenu>
			<DropdownMenuTrigger asChild>{children}</DropdownMenuTrigger>
			<DropdownMenuContent align="end">
				<DropdownMenuGroup>
					<DropdownMenuItem
						onClick={() => {
							setActivePopover(ACTIVE_POPOVER_KIND.rename)
						}}
					>
						Rename link
					</DropdownMenuItem>
					<DropdownMenuItem
						onClick={() => {
							setActivePopover(ACTIVE_POPOVER_KIND.setExpiration)
						}}
					>
						Configure expiration
					</DropdownMenuItem>
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
		</DropdownMenu>
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
