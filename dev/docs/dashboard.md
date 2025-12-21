# Context

- tech stack: react, tailwindcss, shadcn, tanstack router, tanstack table, react-query, jotai
- `@/` maps to `src/`

# Key files/directories

- `src/components`: reusable React components
- `src/lib`: reusable util code
- `src/routes`: tanstack file routes
- `src/vfs`: virtual file system related logic
- `src/user`: user logic and ui components
- `src/directories`: directory related logic and ui components

# Guidelines

- ALWAYS use absolute import using `@/`, unless the file is in the same directory.
- ALWAYS use kebab-case for file names.
- ALWAYS use bun to run scripts
- No testing is in place right now, so skip that for now.
- Use biome for linting, formatting, and import ordering.
- Don't write test/docs unless asked.
- Put reusable code in their corresponding directories (eg `src/components`)
- Read `package.json` for available scripts.
- Always run lint and typecheck for files you modify before replying.
- Data fetching: use `src/lib/api.ts` (`fetchApi`) with arktype schemas; wrap queries/mutations with `queryOptions`/`mutationOptions` and prefer `atomWithQuery`/`atomFamily` for wiring.
- Auth/account context: account-scoped calls must only run when `currentAccountAtom` is set; use the `_authenticated` layout for gating and redirects.
- Routing: follow TanStack file-route conventions; `src/routeTree.gen.ts` is generated and should not be edited manually.
- UI: prefer shadcn components from `src/components/ui`, app-specific components in `src/components`, and shared helpers from `src/lib/utils` (`cn`).
- Error handling: use `defaultOnError` for user-facing failures and keep error mapping in `src/lib/error.ts`.
