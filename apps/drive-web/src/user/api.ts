import { queryOptions } from "@tanstack/react-query"
import { atomWithQuery } from "jotai-tanstack-query"
import { fetchApi } from "../lib/api"
import { User } from "./user"

export const currentUserQuery = queryOptions({
	queryKey: ["currentUser"],
	queryFn: async () =>
		fetchApi("GET", "/users/me", {
			returns: User,
		}).then(([_, result]) => result),
})

export const currentUserAtom = atomWithQuery(() => currentUserQuery)
