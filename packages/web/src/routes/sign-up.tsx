import { useMutation } from "@tanstack/react-query"
import { createFileRoute, useNavigate } from "@tanstack/react-router"
import { GalleryVerticalEnd } from "lucide-react"
import type React from "react"
import { Button } from "@/components/ui/button"
import {
	Card,
	CardContent,
	CardDescription,
	CardHeader,
	CardTitle,
} from "@/components/ui/card"
import {
	Field,
	FieldDescription,
	FieldError,
	FieldGroup,
	FieldLabel,
} from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { type AuthErrorCode, authClient } from "../auth-client"

export const Route = createFileRoute("/sign-up")({
	component: SignupPage,
})

type SignUpParams = {
	displayName: string
	email: string
	password: string
	confirmPassword: string
}

class PasswordMismatchError extends Error {
	constructor() {
		super("Passwords do not match")
	}
}

class BetterAuthError extends Error {
	constructor(public readonly errorCode: AuthErrorCode) {
		super(`better-auth error: ${errorCode}`)
	}
}

function SignupPage() {
	return (
		<div className="bg-background flex min-h-svh flex-col items-center justify-center gap-6 p-6 md:p-10">
			<div className="flex w-full max-w-xl flex-col gap-6">
				<a
					href="#"
					className="flex items-center gap-2 self-center font-medium text-xl"
				>
					<div className="bg-primary text-primary-foreground flex size-6 items-center justify-center rounded-md">
						<GalleryVerticalEnd className="size-4" />
					</div>
					Drexa
				</a>
				<SignUpFormContainer>
					<SignupForm />
				</SignUpFormContainer>
			</div>
		</div>
	)
}

function SignUpFormContainer({ children }: React.PropsWithChildren) {
	return (
		<div className="flex flex-col gap-6">
			<Card>
				<CardHeader className="text-center">
					<CardTitle className="text-xl">
						Create your account
					</CardTitle>
					<CardDescription>
						Enter your email below to create your account
					</CardDescription>
				</CardHeader>
				<CardContent>{children}</CardContent>
			</Card>
		</div>
	)
}

function SignupForm() {
	const navigate = useNavigate()
	const {
		mutate: signUp,
		isPending,
		error: signUpError,
	} = useMutation({
		mutationFn: async (data: SignUpParams) => {
			if (data.password !== data.confirmPassword) {
				throw new PasswordMismatchError()
			}
			const { data: signUpData, error } = await authClient.signUp.email({
				name: data.displayName,
				email: data.email,
				password: data.password,
			})
			if (error) {
				throw new BetterAuthError(error.code as AuthErrorCode)
			}
			return signUpData
		},
		onSuccess: () => {
			navigate({
				to: "/",
				replace: true,
			})
		},
	})

	const handleSubmit = (event: React.FormEvent<HTMLFormElement>) => {
		event.preventDefault()
		const formData = new FormData(event.currentTarget)
		signUp({
			displayName: formData.get("displayName") as string,
			email: formData.get("email") as string,
			password: formData.get("password") as string,
			confirmPassword: formData.get("confirmPassword") as string,
		})
	}

	return (
		<form onSubmit={handleSubmit}>
			<FieldGroup>
				<Field>
					<FieldLabel htmlFor="name">Your Name</FieldLabel>
					<Input
						name="displayName"
						type="text"
						placeholder="John Doe"
						required
					/>
					<FieldDescription>
						This is how you will be referred to on Drexa. You can
						change this later.
					</FieldDescription>
				</Field>
				<Field>
					<FieldLabel htmlFor="email">Email</FieldLabel>
					<Input
						name="email"
						type="email"
						placeholder="m@example.com"
						required
					/>
					{signUpError instanceof BetterAuthError &&
					signUpError.errorCode ===
						authClient.$ERROR_CODES.INVALID_EMAIL ? (
						<FieldError>Invalid email address</FieldError>
					) : null}
				</Field>
				<Field>
					<Field className="grid grid-rows-2 md:grid-rows-1 md:grid-cols-2 gap-4">
						<Field>
							<FieldLabel htmlFor="password">Password</FieldLabel>
							<Input name="password" type="password" required />
						</Field>
						<Field>
							<FieldLabel htmlFor="confirm-password">
								Confirm Password
							</FieldLabel>
							<Input
								name="confirmPassword"
								type="password"
								required
							/>
						</Field>
					</Field>
					{signUpError instanceof PasswordMismatchError ? (
						<FieldError>Passwords do not match</FieldError>
					) : (
						<FieldDescription>
							Must be at least 8 characters long.
						</FieldDescription>
					)}
				</Field>
				<Field>
					<Button
						type="submit"
						loading={isPending}
						disabled={isPending}
					>
						{isPending ? "Creating account…" : "Create Account"}
					</Button>
					<FieldDescription className="text-center">
						Already have an account? <a href="#">Sign in</a>
					</FieldDescription>
				</Field>
			</FieldGroup>
		</form>
	)
}
