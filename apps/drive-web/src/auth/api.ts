import { mutationOptions } from "@tanstack/react-query"
import { type } from "arktype"
import { accountsQuery } from "../account/api"
import { fetchApi } from "../lib/api"
import { currentUserQuery } from "../user/api"
import { User } from "../user/user"

const LoginResponseSchema = type({
	user: User,
})

export const loginMutation = mutationOptions({
	mutationFn: async (data: { email: string; password: string }) => {
		const [_, result] = await fetchApi("POST", "/auth/login", {
			body: JSON.stringify({
				...data,
				tokenDelivery: "cookie",
			}),
			returns: LoginResponseSchema,
		})
		return result
	},
	onSuccess: (data, _, __, context) => {
		context.client.setQueryData(currentUserQuery.queryKey, data.user)
		context.client.invalidateQueries(accountsQuery)
	},
})
