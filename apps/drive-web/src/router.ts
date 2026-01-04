import { createRouter } from "@tanstack/react-router"
import { queryClient } from "./query-client"
import { routeTree } from "./routeTree.gen"

export type RouterContext = {
	queryClient: typeof queryClient
}

export const router = createRouter({
	routeTree,
	context: {
		queryClient,
	},
})
