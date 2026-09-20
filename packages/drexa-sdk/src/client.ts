import { createCatalog } from "./catalog"
import {
	Account,
	type CreateShareInput,
	type CreateUploadInput,
	Drive,
	DriveInfo,
	type DriveScope,
	type LoginInput,
	LoginResponse,
	Organization,
	type RefreshTokenInput,
	Share,
	type SignUpInput,
	SignUpResponse,
	Tokens,
	type UpdateShareInput,
	Upload,
	User,
} from "./models"
import {
	type ClientOptions,
	type RequestOptions,
	segment,
	Transport,
} from "./transport"

export class DrexaClient {
	private readonly transport: Transport

	constructor(options: ClientOptions) {
		this.transport = new Transport(options)
	}

	readonly auth = {
		login: (input: LoginInput, options?: RequestOptions) =>
			this.transport.json(
				"POST",
				this.transport.url("/auth/login"),
				LoginResponse,
				input,
				options,
			),
		signUp: (input: SignUpInput, options?: RequestOptions) =>
			this.transport.json(
				"POST",
				this.transport.url("/accounts"),
				SignUpResponse,
				input,
				options,
			),
		refresh: (input: RefreshTokenInput, options?: RequestOptions) =>
			this.transport.json(
				"POST",
				this.transport.url("/auth/tokens"),
				Tokens,
				input,
				options,
			),
	}

	readonly users = {
		me: (options?: RequestOptions) =>
			this.transport.json(
				"GET",
				this.transport.url("/users/me"),
				User,
				undefined,
				options,
			),
	}

	readonly accounts = {
		list: (options?: RequestOptions) =>
			this.transport.json(
				"GET",
				this.transport.url("/accounts"),
				Account.array(),
				undefined,
				options,
			),
	}

	readonly organizations = {
		list: (options?: RequestOptions) =>
			this.transport.json(
				"GET",
				this.transport.url("/users/me/organizations"),
				Organization.array(),
				undefined,
				options,
			),
		get: (slug: string, options?: RequestOptions) =>
			this.transport.json(
				"GET",
				this.transport.url(`/organizations/${segment(slug)}`),
				Organization,
				undefined,
				options,
			),
	}

	readonly drives = {
		list: (orgSlug: string, options?: RequestOptions) =>
			this.transport.json(
				"GET",
				this.transport.url(`/${segment(orgSlug)}/drives`),
				Drive.array(),
				undefined,
				options,
			),
	}

	drive(scope: DriveScope) {
		const { orgSlug, driveId } = scope
		const base = `/${segment(orgSlug)}/drives/${segment(driveId)}`
		const transport = this.transport
		const url = (path = "") => transport.url(`${base}${path}`)
		return {
			...createCatalog(transport, base),
			get: (options?: RequestOptions) =>
				transport.json("GET", url(), DriveInfo, undefined, options),
			uploads: {
				create: (input: CreateUploadInput, options?: RequestOptions) =>
					transport.json(
						"POST",
						url("/uploads"),
						Upload,
						input,
						options,
					),
				/** Send bytes through the API. Browser progress handling can use contentUrl() with XHR. */
				putContent: async (
					id: string,
					content: BodyInit,
					options?: RequestOptions,
				): Promise<void> => {
					const headers = new Headers(options?.headers)
					if (!headers.has("Content-Type"))
						headers.set("Content-Type", "application/octet-stream")
					await transport.send(
						"PUT",
						url(`/uploads/${segment(id)}/content`),
						content,
						{ ...options, headers },
					)
				},
				contentUrl: (id: string) =>
					url(`/uploads/${segment(id)}/content`),
				complete: (id: string, options?: RequestOptions) =>
					transport.json(
						"PATCH",
						url(`/uploads/${segment(id)}`),
						Upload,
						{ status: "completed" },
						options,
					),
			},
			shares: {
				get: (id: string, options?: RequestOptions) =>
					transport.json(
						"GET",
						url(`/shares/${segment(id)}`),
						Share,
						undefined,
						options,
					),
				create: (input: CreateShareInput, options?: RequestOptions) =>
					transport.json(
						"POST",
						url("/shares"),
						Share,
						input,
						options,
					),
				update: (
					id: string,
					input: UpdateShareInput,
					options?: RequestOptions,
				) =>
					transport.json(
						"PATCH",
						url(`/shares/${segment(id)}`),
						Share,
						input,
						options,
					),
				delete: (id: string, options?: RequestOptions) =>
					transport.empty(
						"DELETE",
						url(`/shares/${segment(id)}`),
						options,
					),
			},
		}
	}

	share(shareId: string) {
		return createCatalog(this.transport, `/shares/${segment(shareId)}`)
	}
}
export type DriveClient = ReturnType<DrexaClient["drive"]>
