import { QueryClient } from "@tanstack/react-query"
import { defaultOnError } from "./lib/error"

export const queryClient = new QueryClient({
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
