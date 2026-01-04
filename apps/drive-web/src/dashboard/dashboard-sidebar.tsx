import { useMutation, useQuery } from "@tanstack/react-query"
import {
	getRouteApi,
	Link,
	useParams,
	useRouterState,
} from "@tanstack/react-router"
import { useAtom, useAtomValue, useSetAtom } from "jotai"
import {
	CircleXIcon,
	FilesIcon,
	FolderInputIcon,
	HouseIcon,
	LogOutIcon,
	ScissorsIcon,
	SettingsIcon,
	User2Icon,
} from "lucide-react"
import { toast } from "sonner"
import { Card, CardFooter, CardHeader, CardTitle } from "@/components/ui/card"
import {
	DropdownMenu,
	DropdownMenuContent,
	DropdownMenuItem,
	DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import {
	Sidebar,
	SidebarContent,
	SidebarFooter,
	SidebarGroup,
	SidebarHeader,
	SidebarMenu,
	SidebarMenuButton,
	SidebarMenuItem,
} from "@/components/ui/sidebar"
import { formatError } from "@/lib/error"
import { currentDriveAtom } from "@/drive/drive"
import { listOrganizationDrivesQuery } from "@/organization/api"
import { useCurrentOrganization } from "@/organization/context"
import { moveDirectoryItemsMutationOptions } from "@/vfs/api"
import { Button } from "../components/ui/button"
import { LoadingSpinner } from "../components/ui/loading-spinner"
import { clearCutItemsAtom, cutItemsAtom } from "../files/store"
import { backgroundTaskProgressAtom } from "./state"

export function DashboardSidebar() {
	return (
		<Sidebar variant="inset" collapsible="icon">
			<SidebarHeader>
				<SidebarMenu>
					<SidebarMenuItem>
						<UserMenu />
					</SidebarMenuItem>
				</SidebarMenu>
			</SidebarHeader>
			<SidebarContent>
				<SidebarGroup>
					<MainSidebarMenu />
				</SidebarGroup>
			</SidebarContent>
			<SidebarFooter>
				<SidebarMenu>
					<CutItemsCard />
					<BackgroundTaskProgressItem />
				</SidebarMenu>
			</SidebarFooter>
		</Sidebar>
	)
}

function MainSidebarMenu() {
	const org = useCurrentOrganization()
	const matchingRoute = getRouteApi(
		"/_authenticated/$orgSlug/_sidebar-layout/",
	)
	const isActive = useRouterState({
		select: (state) =>
			state.matches.some((match) => match.routeId === matchingRoute.id),
	})

	return (
		<SidebarMenu>
			<SidebarMenuItem>
				<SidebarMenuButton asChild isActive={isActive}>
					<Link to="/$orgSlug" params={{ orgSlug: org.slug }}>
						<HouseIcon />
						<span>Home</span>
					</Link>
				</SidebarMenuButton>
			</SidebarMenuItem>
			<AllFilesItem />
		</SidebarMenu>
	)
}

function AllFilesItem() {
	const org = useCurrentOrganization()
	const matchingRoute = getRouteApi(
		"/_authenticated/$orgSlug/_sidebar-layout/drives/$driveId",
	)
	const isActive = useRouterState({
		select: (state) =>
			state.matches.some((match) => match.routeId === matchingRoute.id),
	})

	const { data: drives } = useQuery(listOrganizationDrivesQuery(org.slug))
	const drive = drives?.at(0)

	if (!drive) return null

	return (
		<SidebarMenuItem>
			<SidebarMenuButton asChild isActive={isActive}>
				<Link
					to="/$orgSlug/drives/$driveId"
					params={{ orgSlug: org.slug, driveId: drive.id }}
				>
					<FilesIcon />
					<span>All Files</span>
				</Link>
			</SidebarMenuButton>
		</SidebarMenuItem>
	)
}

function BackgroundTaskProgressItem() {
	const backgroundTaskProgress = useAtomValue(backgroundTaskProgressAtom)

	if (!backgroundTaskProgress) return null

	return (
		<SidebarMenuItem className="flex items-center gap-2 opacity-80 text-sm">
			<LoadingSpinner />
			{backgroundTaskProgress.label}
		</SidebarMenuItem>
	)
}

/**
 * Displays the number of cut items and allows the user to perform actions on them, such as moving them to a target directory.
 * Visible when there are cut items.
 */
function CutItemsCard() {
	const { directoryId } = useParams({ strict: false })
	const [cutItems, setCutItems] = useAtom(cutItemsAtom)
	const clearCutItems = useSetAtom(clearCutItemsAtom)
	const setBackgroundTaskProgress = useSetAtom(backgroundTaskProgressAtom)

	const org = useCurrentOrganization()
	const drive = useAtomValue(currentDriveAtom)
	const moveDirectoryItemsMutation = moveDirectoryItemsMutationOptions({
		org,
		drive,
	})

	const { mutate: moveItems } = useMutation({
		...moveDirectoryItemsMutation,
		onMutate: () => {
			setBackgroundTaskProgress({
				label: "Moving items…",
			})
			clearCutItems()
			return { cutItems }
		},
		onError: (error, _variables, context) => {
			if (context?.cutItems) {
				setCutItems(context.cutItems)
			}
			toast.error("Failed to move items", {
				description: formatError(error),
			})
		},
		onSuccess: () => {
			toast.success("Items moved")
		},
		onSettled: () => {
			setBackgroundTaskProgress(null)
		},
	})

	if (cutItems.length === 0) return null

	const moveCutItems = () => {
		if (directoryId) {
			moveItems({
				targetDirectory: directoryId,
				items: cutItems,
			})
		}
	}

	return (
		<SidebarMenuItem>
			<Card className="p-0 gap-0 rounded-md overflow-clip">
				<CardHeader className="px-3.5 py-1.5! gap-0 border-b border-b-primary-foreground/10 bg-primary text-primary-foreground">
					<CardTitle className="p-0 m-0 text-xs uppercase">
						<div className="flex items-center gap-1.5">
							<ScissorsIcon size={16} /> {cutItems.length} Cut
							Items
						</div>
					</CardTitle>
				</CardHeader>
				<CardFooter className="p-1 flex flex-col">
					<Button
						size="sm"
						variant="ghost"
						className="w-full justify-start transition-none"
						disabled={!directoryId}
						onClick={moveCutItems}
					>
						<FolderInputIcon size={16} />
						Move items here
					</Button>
					<Button
						size="sm"
						variant="ghost"
						className="w-full justify-start transition-none"
						onClick={() => clearCutItems()}
					>
						<CircleXIcon size={16} />
						Clear
					</Button>
				</CardFooter>
			</Card>
		</SidebarMenuItem>
	)
}

function UserMenu() {
	function handleSignOut() {}

	return (
		<DropdownMenu>
			<DropdownMenuTrigger asChild>
				<SidebarMenuButton size="lg" asChild>
					<a href="/">
						<div className="bg-sidebar-primary text-sidebar-primary-foreground flex aspect-square size-8 items-center justify-center rounded-lg">
							<User2Icon className="size-4" />
						</div>
						<div className="grid flex-1 text-left text-sm leading-tight">
							<span className="truncate font-medium">
								Acme Inc
							</span>
							<span className="truncate text-xs">Enterprise</span>
						</div>
					</a>
				</SidebarMenuButton>
			</DropdownMenuTrigger>
			<DropdownMenuContent className="w-64" align="start" side="bottom">
				<DropdownMenuItem>
					<SettingsIcon />
					Settings
				</DropdownMenuItem>
				<DropdownMenuItem onClick={handleSignOut}>
					<LogOutIcon />
					Log out
				</DropdownMenuItem>
			</DropdownMenuContent>
		</DropdownMenu>
	)
}
