import { queryOptions } from "@tanstack/react-query"
import { fetchApi } from "@/lib/api"
import { Account } from "./account"

export const accountsQuery = queryOptions({
	queryKey: ["accounts"],
	queryFn: async () =>
		fetchApi("GET", "/accounts", {
			returns: Account.array(),
		}).then(([_, result]) => result),
})
