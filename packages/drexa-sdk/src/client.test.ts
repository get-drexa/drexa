import { describe, expect, test } from "bun:test"
import * as sdk from "./index"
import {
	ApiError,
	type ClientOptions,
	DrexaClient,
	type DriveListItem,
} from "./index"

const timestamp = "2026-01-05T01:25:37Z"
const drive = {
	id: "drive1",
	name: "Personal",
	createdAt: timestamp,
	updatedAt: timestamp,
	storageUsageBytes: 0,
	storageQuotaBytes: 1024,
}
const directory = {
	id: "dir1",
	kind: "directory",
	name: "Documents",
	createdAt: timestamp,
	updatedAt: timestamp,
	path: [{ id: "dir1", name: "Documents" }],
}
const scope = { orgSlug: "my", driveId: "drive1" }

function fixture(
	respond: (request: Request) => Response | Promise<Response>,
	options: Partial<ClientOptions> = {},
) {
	const requests: Request[] = []
	const requestOptions: (RequestInit | undefined)[] = []
	const client = new DrexaClient({
		baseUrl: "https://example.test/prefix/api/",
		...options,
		fetch: async (input, init) => {
			requestOptions.push(init)
			const request = new Request(input, init)
			requests.push(request)
			return respond(request)
		},
	})
	return { client, requests, requestOptions }
}

describe("Drexa SDK", () => {
	test("public runtime exports contain no ArkType definitions", () => {
		expect(Object.keys(sdk).sort()).toEqual(["ApiError", "DrexaClient"])
	})

	test("parses drive list dates and root directory IDs", async () => {
		const { client, requests } = fixture(() =>
			Response.json([{ ...drive, rootDir: "root1" }]),
		)
		const result: DriveListItem[] = await client.drives.list("my")
		expect(result[0]?.createdAt).toEqual(new Date(timestamp))
		expect(result[0]?.rootDir).toBe("root1")
		expect(requests[0]?.url).toBe(
			"https://example.test/prefix/api/my/drives",
		)
	})

	test("requires rootDir in drive list responses", async () => {
		const { client } = fixture(() => Response.json([drive]))
		await expect(client.drives.list("my")).rejects.toThrow("rootDir")
	})

	test("validates drive detail responses and malformed payloads", async () => {
		const { client } = fixture(() => Response.json(drive))
		await expect(client.drive(scope).get()).rejects.toThrow("rootDirId")
		const invalid = fixture(() =>
			Response.json({ ...drive, createdAt: "invalid" }),
		)
		await expect(invalid.client.drive(scope).get()).rejects.toThrow()
	})

	test("rotates bearer tokens and preserves configured/request headers", async () => {
		let token = "first"
		const { client, requests, requestOptions } = fixture(
			() => Response.json([]),
			{
				accessToken: async () => token,
				headers: { "X-Client": "cli" },
			},
		)
		await client.drives.list("my", { headers: { "X-Request": "one" } })
		token = "second"
		await client.drives.list("my")
		expect(requests[0]?.headers.get("Authorization")).toBe("Bearer first")
		expect(requests[1]?.headers.get("Authorization")).toBe("Bearer second")
		expect(requests[0]?.headers.get("X-Client")).toBe("cli")
		expect(requests[0]?.headers.get("X-Request")).toBe("one")
		expect(requestOptions[0]?.credentials).toBe("omit")
		expect(requests[0]?.headers.has("Content-Type")).toBe(false)
	})

	test("cookie login and body-token login preserve their delivery modes", async () => {
		const { client, requestOptions } = fixture(
			async (request) => {
				expect(await request.json()).toEqual({
					email: "a@b.test",
					password: "secret",
					tokenDelivery: "cookie",
				})
				return Response.json({
					user: { id: "u", displayName: "A", email: "a@b.test" },
					organizations: [],
				})
			},
			{ credentials: "include" },
		)
		await client.auth.login({
			email: "a@b.test",
			password: "secret",
			tokenDelivery: "cookie",
		})
		expect(requestOptions[0]?.credentials).toBe("include")
		const body = fixture(() =>
			Response.json({
				user: { id: "u", displayName: "A", email: "a@b.test" },
				organizations: [],
				accessToken: "access",
				refreshToken: "refresh",
			}),
		)
		expect(
			(
				await body.client.auth.login({
					email: "a@b.test",
					password: "secret",
					tokenDelivery: "body",
				})
			).accessToken,
		).toBe("access")
	})

	test("encodes scope identifiers and pagination without changing the API base", async () => {
		const { client, requests } = fixture(() => Response.json({ items: [] }))
		await client
			.drive({ orgSlug: "team name", driveId: "drive/one" })
			.directories.list("dir?one", {
				orderBy: "name",
				direction: "desc",
				limit: 20,
				cursor: "a+b/c==&next=bad",
			})
		const url = new URL(requests[0]?.url ?? "")
		expect(url.pathname).toBe(
			"/prefix/api/team%20name/drives/drive%2Fone/directories/dir%3Fone/content",
		)
		expect(url.searchParams.get("cursor")).toBe("a+b/c==&next=bad")
		expect(url.searchParams.get("dir")).toBe("desc")
		expect(url.searchParams.get("limit")).toBe("20")
		expect(() => client.drive({ orgSlug: "..", driveId: "x" })).toThrow()
	})

	test("creates directories using the backend parentID field", async () => {
		const { client } = fixture(async (request) => {
			expect(request.method).toBe("POST")
			expect(await request.json()).toEqual({
				parentID: "root",
				name: "Documents",
			})
			return Response.json(directory)
		})
		const result = await client
			.drive(scope)
			.directories.create({ parentId: "root", name: "Documents" })
		expect(result.path[0]?.name).toBe("Documents")
	})

	test("keeps public share access separate from drive scope", async () => {
		const { client, requests } = fixture(() => Response.json(directory))
		await client.share("share1").directories.get("root")
		expect(requests[0]?.url).toBe(
			"https://example.test/prefix/api/shares/share1/directories/root?include=path",
		)
		expect(client.share("share1").files.contentUrl("file1")).toBe(
			"https://example.test/prefix/api/shares/share1/files/file1/content",
		)
	})

	test("permanent deletion accepts empty 204 responses; empty batches do not request", async () => {
		const { client, requests } = fixture(
			() => new Response(null, { status: 204 }),
		)
		const scoped = client.drive(scope)
		await scoped.files.delete("file1")
		await scoped.directories.deleteMany(["dir1", "dir2"])
		await scoped.shares.delete("share1")
		await scoped.files.deleteMany([])
		expect(await scoped.files.trashMany([])).toEqual([])
		expect(
			await scoped.directories.moveItems("dir1", { items: [] }),
		).toEqual({ items: [], moved: [], conflicts: [], errors: [] })
		expect(requests).toHaveLength(3)
		expect(new URL(requests[1]?.url ?? "").searchParams.get("id")).toBe(
			"dir1,dir2",
		)
	})

	test("upload bytes stay raw and completion accepts empty uploadUrl", async () => {
		const { client } = fixture(async (request) => {
			if (request.method === "PUT") {
				expect(await request.text()).toBe("file bytes")
				expect(request.headers.get("Content-Type")).toBe("text/plain")
				return new Response(null, { status: 204 })
			}
			expect(await request.json()).toEqual({ status: "completed" })
			return Response.json({
				id: "upload1",
				status: "completed",
				uploadUrl: "",
			})
		})
		await client.drive(scope).uploads.putContent("upload1", "file bytes", {
			headers: { "Content-Type": "text/plain" },
		})
		expect(
			(await client.drive(scope).uploads.complete("upload1")).status,
		).toBe("completed")
	})

	test("distinguishes clearing share expiry from omitting it", async () => {
		const bodies: unknown[] = []
		const { client } = fixture(async (request) => {
			bodies.push(await request.json())
			return Response.json({
				id: "s",
				expiresAt: null,
				createdAt: timestamp,
				updatedAt: timestamp,
			})
		})
		await client.drive(scope).shares.update("s", {})
		await client.drive(scope).shares.update("s", { expiresAt: null })
		await client
			.drive(scope)
			.shares.update("s", { expiresAt: new Date(timestamp) })
		expect(bodies).toEqual([
			{},
			{ expiresAt: null },
			{ expiresAt: new Date(timestamp).toISOString() },
		])
	})

	test("reports structured and plain-text API errors", async () => {
		const structured = fixture(() =>
			Response.json({ error: "No access" }, { status: 403 }),
		)
		await expect(structured.client.users.me()).rejects.toMatchObject({
			name: "ApiError",
			status: 403,
			message: "No access",
		})
		const plain = fixture(() => new Response("Not Found", { status: 404 }))
		try {
			await plain.client.users.me()
			throw new Error("Expected rejection")
		} catch (error) {
			expect(error).toBeInstanceOf(ApiError)
			expect(error).toMatchObject({ status: 404, message: "Not Found" })
		}
	})

	test("passes cancellation through without wrapping the error", async () => {
		const controller = new AbortController()
		controller.abort()
		const { client } = fixture((request) => {
			request.signal.throwIfAborted()
			return Response.json([])
		})
		await expect(
			client.drives.list("my", { signal: controller.signal }),
		).rejects.toMatchObject({ name: "AbortError" })
	})
})
