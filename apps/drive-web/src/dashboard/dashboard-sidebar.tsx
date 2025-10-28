import { api } from "@fileone/convex/api"
import { Link, useLocation } from "@tanstack/react-router"
import { useQuery as useConvexQuery } from "convex/react"
import { useAtomValue } from "jotai"
import {
	ClockIcon,
	FilesIcon,
	LogOutIcon,
	SettingsIcon,
	TrashIcon,
	User2Icon,
} from "lucide-react"
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
import { LoadingSpinner } from "../components/ui/loading-spinner"
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
					<BackgroundTaskProgressItem />
				</SidebarMenu>
			</SidebarFooter>
		</Sidebar>
	)
}

function MainSidebarMenu() {
	const location = useLocation()

	const isActive = (path: string) => {
		if (path === "/") {
			return location.pathname === "/"
		}
		return location.pathname.startsWith(path)
	}

	return (
		<SidebarMenu>
			<SidebarMenuItem>
				<SidebarMenuButton asChild isActive={isActive("/recent")}>
					<Link to="/recent">
						<ClockIcon />
						<span>Recent</span>
					</Link>
				</SidebarMenuButton>
			</SidebarMenuItem>
			<AllFilesItem />
			<TrashItem />
		</SidebarMenu>
	)
}

function AllFilesItem() {
	const location = useLocation()
	const rootDirectory = useConvexQuery(api.files.fetchRootDirectory)

	if (!rootDirectory) return null

	return (
		<SidebarMenuItem>
			<SidebarMenuButton
				asChild
				isActive={location.pathname.startsWith("/directories")}
			>
				<Link to={`/directories/${rootDirectory._id}`}>
					<FilesIcon />
					<span>All Files</span>
				</Link>
			</SidebarMenuButton>
		</SidebarMenuItem>
	)
}

function TrashItem() {
	const location = useLocation()
	const rootDirectory = useConvexQuery(api.files.fetchRootDirectory)

	if (!rootDirectory) return null

	return (
		<SidebarMenuItem>
			<SidebarMenuButton
				asChild
				isActive={location.pathname.startsWith("/trash/directories")}
			>
				<Link to={`/trash/directories/${rootDirectory._id}`}>
					<TrashIcon />
					<span>Trash</span>
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
