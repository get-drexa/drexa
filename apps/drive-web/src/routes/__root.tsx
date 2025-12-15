import "@/styles/globals.css"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { createRootRoute, Outlet } from "@tanstack/react-router"
import { Provider } from "jotai"
import { useHydrateAtoms } from "jotai/utils"
import { queryClientAtom } from "jotai-tanstack-query"
import type React from "react"
import { Toaster } from "@/components/ui/sonner"
import { defaultOnError } from "@/lib/error"
import { useKeyboardModifierListener } from "@/lib/keyboard"

export const Route = createRootRoute({
	component: RootLayout,
})

const queryClient = new QueryClient({
	defaultOptions: {
		queries: {
			throwOnError: false,
		},
		mutations: {
			onError: defaultOnError,
			throwOnError: false,
		},
	},
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
