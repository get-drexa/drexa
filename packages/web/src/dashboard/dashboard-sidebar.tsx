import { api } from "@fileone/convex/_generated/api"
import { Link, useLocation } from "@tanstack/react-router"
import { useAuth } from "@workos-inc/authkit-react"
import { useQuery as useConvexQuery } from "convex/react"
import {
	ChevronDownIcon,
	FilesIcon,
	HomeIcon,
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
	SidebarHeader,
	SidebarMenu,
	SidebarMenuButton,
	SidebarMenuItem,
	SidebarRail,
} from "@/components/ui/sidebar"

export function DashboardSidebar() {
	return (
		<Sidebar variant="inset" collapsible="icon">
			<SidebarHeader>
				<SidebarMenu>
					<SidebarMenuItem>
						<UserMenu />
					</SidebarMenuItem>
				</SidebarMenu>
				<MainSidebarMenu />
			</SidebarHeader>
			<SidebarRail />
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
				<SidebarMenuButton asChild isActive={isActive("/")}>
					<Link to="/">
						<HomeIcon />
						<span>Home</span>
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
				isActive={location.pathname.startsWith(
					`/directories/${rootDirectory._id}`,
				)}
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
				isActive={location.pathname.startsWith(
					`/trash/directories/${rootDirectory._id}`,
				)}
			>
				<Link to={`/trash/directories/${rootDirectory._id}`}>
					<TrashIcon />
					<span>Trash</span>
				</Link>
			</SidebarMenuButton>
		</SidebarMenuItem>
	)
}

function UserMenu() {
	const { signOut } = useAuth()

	function handleSignOut() {
		signOut()
	}

	return (
		<DropdownMenu>
			<DropdownMenuTrigger asChild>
				<SidebarMenuButton className="w-fit px-1.5 group-data-[collapsible=icon]:px-1.5! data-[state=open]:bg-sidebar-accent">
					<div className="bg-sidebar-primary text-sidebar-primary-foreground flex aspect-square size-5 items-center justify-center rounded-md">
						<User2Icon className="size-3" />
					</div>
					<span className="truncate font-medium">Kenneth</span>
					<ChevronDownIcon className="opacity-50" />
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
