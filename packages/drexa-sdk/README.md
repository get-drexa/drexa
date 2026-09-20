# drexa-sdk

Framework-independent TypeScript client for the Drexa API. ArkType validates responses internally; the public API exports inferred types, not runtime schemas. Response timestamps are parsed into `Date` objects.

Inputs, scope, and request options use plain TypeScript types without runtime schema validation. The backend validates request values.

## Workspace setup

The SDK is currently a private source package, not a published npm package. When integrating a workspace app, add this to its dependencies and run `bun install` from the repository root:

```json
{
	"dependencies": {
		"drexa-sdk": "workspace:*"
	}
}
```

Use a TypeScript-aware runtime or bundler, such as Bun or Vite. The SDK uses standard Fetch APIs and has no React, React Query, or Jotai dependencies.

## Browser authentication

Provide an absolute URL including the API path. Cookie authentication requires `credentials: "include"` and matching backend cookie/CORS settings.

```ts
import { ApiError, DrexaClient, type DirectoryItem } from "drexa-sdk"

const sdk = new DrexaClient({
	baseUrl: "http://localhost:8080/api",
	credentials: "include",
})

const session = await sdk.auth.login({
	email: "you@example.com",
	password: "your-password",
	tokenDelivery: "cookie",
})

console.log(session.user, session.organizations)
const me = await sdk.users.me()
```

To register instead, call `sdk.auth.signUp()` with the same fields plus `displayName`. Registration also authenticates the new user.

## Bearer authentication / CLI

Use `tokenDelivery: "body"` to receive tokens. The SDK does not store tokens or automatically refresh them. Supply the current access token directly or through a callback evaluated before each request.

```ts
let accessToken: string | undefined
let refreshToken: string | undefined

const tokenClient = new DrexaClient({
	baseUrl: "http://localhost:8080/api",
	accessToken: () => accessToken,
})

const login = await tokenClient.auth.login({
	email: "you@example.com",
	password: "your-password",
	tokenDelivery: "body",
})

accessToken = login.accessToken
refreshToken = login.refreshToken

// When your application needs to refresh the session:
if (!refreshToken) throw new Error("Missing refresh token")
const refreshed = await tokenClient.auth.refresh({ refreshToken })
accessToken = refreshed.accessToken
refreshToken = refreshed.refreshToken
```

Credentials default to `"omit"`. Server-side Fetch does not provide a browser cookie jar; use bearer tokens for CLI clients.

## Select a drive and browse

The following examples continue with the authenticated `sdk` above. `my` resolves to the current user's personal organization. A scoped drive client retains explicit organization/drive IDs; there is no global selected-drive state.

```ts
const organizations = await sdk.organizations.list()
const organization = await sdk.organizations.get("my")
const accounts = await sdk.accounts.list()

const drives = await sdk.drives.list(organization.slug)
const selected = drives[0]
if (!selected) throw new Error("No drive available")

const drive = sdk.drive({ orgSlug: organization.slug, driveId: selected.id })
const info = await drive.get()
const root = await drive.directories.get(info.rootDirId)

console.log(info.createdAt.toISOString(), root.path)

const listing = { orderBy: "name", direction: "asc", limit: 100 } as const
const firstPage = await drive.directories.list(root.id, listing)
const items: DirectoryItem[] = [...firstPage.items]

if (firstPage.nextCursor) {
	const nextPage = await drive.directories.list(root.id, {
		...listing,
		cursor: firstPage.nextCursor,
	})
	items.push(...nextPage.items)
}
```

`drives.list()` returns `Drive[]`; `drive.get()` returns `DriveInfo`, which additionally includes `rootDirId`. `directories.get("root")` also resolves the current scope's root. Keep sorting options unchanged when following a pagination cursor.

## Create, rename, move, and trash

```ts
const folder = await drive.directories.create({ parentId: root.id, name: "Drafts" })
await drive.directories.rename(folder.id, { name: "Documents" })

const archive = await drive.directories.create({ parentId: root.id, name: "Archive" })
const moved = await drive.directories.moveItems(archive.id, { items: [folder.id] })
console.log(moved.moved, moved.conflicts, moved.errors)

await drive.directories.trash(folder.id)
```

`moveItems()` moves files or directories into the target directory; source items must share a parent. Inspect its result for partial failures. Files offer equivalent `rename()` and `trash()` methods. Both collections also provide `trashMany(ids)`.

`delete(id)` and `deleteMany(ids)` permanently delete content and resolve without a value. They are separate from trash operations.

## Upload and download

This flow sends content through the current backend's API and explicitly completes the upload. `putContent()` accepts a Fetch `BodyInit`, including a `Blob`, browser `File`, or string.

```ts
const upload = await drive.uploads.create({ parentId: root.id, name: "hello.txt" })

await drive.uploads.putContent(upload.id, new Blob(["Hello, Drexa!"]), {
	headers: { "Content-Type": "text/plain" },
})
await drive.uploads.complete(upload.id)

// Upload IDs are the target files' public IDs in the current backend.
const file = await drive.files.get(upload.id)
const response = await drive.files.download(file.id)
const content = await response.blob()
```

`files.download()` applies the client's authentication and returns a `Response` for streaming or reading. `files.contentUrl(id)` returns a bare URL for browser links/previews; it cannot attach bearer headers, so use it with cookie authentication or public shares.

Upload progress is UI-specific and stays outside the SDK. For browser XHR, `uploads.contentUrl(id)` gives the API upload endpoint; configure its authentication yourself. The create response also exposes `uploadUrl`. `putContent()` always targets the API, not an external signed URL, and completion may return an empty `uploadUrl`.

## Sharing

```ts
const share = await drive.shares.create({
	items: [file.id],
	expiresAt: new Date(Date.now() + 24 * 60 * 60 * 1000),
})

await drive.shares.update(share.id, { expiresAt: null }) // Remove expiry.
const shares = await drive.files.listShares(file.id, { includesExpired: true })

const publicSdk = new DrexaClient({ baseUrl: "http://localhost:8080/api" })
const shared = publicSdk.share(share.id)
const sharedRoot = await shared.directories.get("root")
const sharedItems = await shared.directories.list(sharedRoot.id)
const sharedDownloadUrl = shared.files.contentUrl(file.id)
```

Omit `expiresAt` from an update to preserve it; pass `null` to clear it. Share creation requires items from the same parent directory. Share clients expose the same file/directory methods, but the server enforces the share's permissions. Use an authenticated client for shares that require account access. Revoke a share with `drive.shares.delete(share.id)`.

## Errors, cancellation, and request configuration

```ts
const controller = new AbortController()

try {
	await drive.directories.list(root.id, {}, {
		signal: controller.signal,
		headers: { "X-Request-ID": "browse-example" },
	})
} catch (error) {
	if (error instanceof ApiError) {
		console.error(error.status, error.message, error.body)
	} else {
		// Validation, network, and cancellation errors retain their original types.
		throw error
	}
}
```

Call `controller.abort()` when the caller no longer needs the request. Request options are the last argument; methods with a separate input/options object take them after it.

Client options also accept default `headers` and a custom `fetch` function for testing or transport integration. Per-request headers override defaults; configured access tokens supply the `Authorization` header. JSON methods set the appropriate JSON headers. Query caching, token persistence, retries, and UI state belong to the consuming application.
