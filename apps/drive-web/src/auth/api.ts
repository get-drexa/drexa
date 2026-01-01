import { mutationOptions } from "@tanstack/react-query"
import { type } from "arktype"
import { Account } from "@/account/account"
import { accountsQuery } from "@/account/api"
import { fetchApi } from "@/lib/api"
import { currentUserQuery } from "@/user/api"
import { User } from "@/user/user"
import { drivesQuery } from "@/drive/api"
import { Drive } from "@/drive/drive"

const LoginResponseSchema = type({
	user: User,
})

const SignUpResponse = type({
	account: Account,
	user: User,
	drive: Drive,
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
		context.client.invalidateQueries(drivesQuery)
	},
})

export const signUpMutation = mutationOptions({
	mutationFn: async (data: {
		displayName: string
		email: string
		password: string
	}) => {
		const [_, result] = await fetchApi("POST", "/accounts", {
			body: JSON.stringify({
				...data,
				tokenDelivery: "cookie",
			}),
			returns: SignUpResponse,
		})
		return result
	},
	onSuccess: (data, _, __, context) => {
		context.client.setQueryData(currentUserQuery.queryKey, data.user)
		context.client.setQueryData(accountsQuery.queryKey, [data.account])
		context.client.setQueryData(drivesQuery.queryKey, [data.drive])
	},
})
