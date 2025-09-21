# Context

- tech stack: react, tailwindcss, shadcn, tanstack router, tanstack table, jotai
- `@/` maps to `src/`

# Key files

- `src/components`: reusable React components
- `src/lib`: reusable util code
- `src/routes`: tanstack file routes

# Guidelines

- ALWAYS use absolute import using `@/`, unless the file is in the same directory.
- ALWAYS use kebab-case for file names
- when importing useQuery/useMutation from convex/react, alias them with useConvexQuery/useConvexMutation
- Do not attempt to create a preview server. There is one running already at port 3001.
- After create a new file route, run `bunx tsr generate`
- No testing is in place right now, so skip that for now.
