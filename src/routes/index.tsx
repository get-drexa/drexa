import { createFileRoute } from "@tanstack/react-router"
import { HomeIcon } from "lucide-react"
import { SidebarTrigger } from "@/components/ui/sidebar"

export const Route = createFileRoute("/")({
	component: Index,
})

function Index() {
	return (
		<>
			<header className="flex py-2 shrink-0 items-center gap-2 border-b px-4">
				<SidebarTrigger className="-ml-1" />
				<div className="flex items-center gap-2">
					<HomeIcon className="h-5 w-5" />
					<h1 className="text-lg font-semibold">Home</h1>
				</div>
			</header>
			<div className="p-6">
				<h1 className="text-2xl font-bold">Welcome to FileOne</h1>
				<p className="text-muted-foreground mt-2">
					Your file management dashboard
				</p>
			</div>
		</>
	)
}
