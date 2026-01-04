import "@/styles/globals.css"
import { QueryClientProvider } from "@tanstack/react-query"
import { createRootRouteWithContext, Outlet } from "@tanstack/react-router"
import { Provider } from "jotai"
import { useHydrateAtoms } from "jotai/utils"
import { queryClientAtom } from "jotai-tanstack-query"
import type React from "react"
import { Toaster } from "@/components/ui/sonner"
import { useKeyboardModifierListener } from "@/lib/keyboard"
import { queryClient } from "@/query-client"
import type { RouterContext } from "@/router"

export const Route = createRootRouteWithContext<RouterContext>()({
	component: RootLayout,
})

function HydrateAtoms({ children }: React.PropsWithChildren) {
	useHydrateAtoms(new Map([[queryClientAtom, queryClient]]))
	return children
}

function RootLayout() {
	useKeyboardModifierListener()

	return (
		<QueryClientProvider client={queryClient}>
			<Provider>
				<HydrateAtoms>
					<Outlet />
					<Toaster />
				</HydrateAtoms>
			</Provider>
		</QueryClientProvider>
	)
}
