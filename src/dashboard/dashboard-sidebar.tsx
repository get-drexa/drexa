import { Link, useLocation } from "@tanstack/react-router"
import { FilesIcon, HomeIcon, User2Icon } from "lucide-react"
import {
	DropdownMenu,
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
		<Sidebar collapsible="icon">
			<SidebarHeader>
				<SidebarMenu>
					<SidebarMenuItem>
						<UserSwitcher />
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
			<SidebarMenuItem>
				<SidebarMenuButton asChild isActive={isActive("/files")}>
					<Link to="/files">
						<FilesIcon />
						<span>All Files</span>
					</Link>
				</SidebarMenuButton>
			</SidebarMenuItem>
		</SidebarMenu>
	)
}

function UserSwitcher() {
	return (
		<DropdownMenu>
			<DropdownMenuTrigger asChild>
				<SidebarMenuButton size="lg" className="w-fit px-1.5">
					<div className="bg-sidebar-primary text-sidebar-primary-foreground flex aspect-square size-8 items-center justify-center rounded-md">
						<User2Icon className="size-4" />
					</div>
					<span>Kenneth</span>
				</SidebarMenuButton>
			</DropdownMenuTrigger>
		</DropdownMenu>
	)
}
