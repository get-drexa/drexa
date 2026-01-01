import { queryOptions } from "@tanstack/react-query"
import { fetchApi } from "@/lib/api"
import { Drive } from "./drive"

export const drivesQuery = queryOptions({
	queryKey: ["drives"],
	queryFn: async () =>
		fetchApi("GET", "/drives", {
			returns: Drive.array(),
		}).then(([_, result]) => result),
})
