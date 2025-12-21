# Project tech stack

frontend: react, shadcn, tailwindcss
backend: go with go fiber as router

# Project structure

This project uses npm workspaces.
- `apps/` - application code
  - `drive-web` - web app for drexa drive, a cloud storage service
  - `backend` - backend that powers drexa (will offer more services beyond drive in the future)
  - `file-proxy` - deprecated go service to support the de
- `packages/` - reusable typescript code
  - `convex` - deprecated convex backend that i am migrating away from
  - `path` - deprecated path util for convex
  - `auth` - deprecated auth package

# General Guidelines

- IGNORE CONVEX. I DO NOT WANT CONVEX CODE ANYMORE.
- For frontend dashboard guidelines, refer to dev/docs/dashboard.md
- For backend guidelines, refer to dev/docs/backend.md
- When commiting, follow conventional commits: `[type]: msg` and clarify in commit body if necessary. keep your commit header to <= 50 characters.
  - `feat:` for new features
  - `fix:` for bug fixes  
  - `docs:` for documentation
  - `refactor:` for code restructuring
  - `style:` for formatting changes
  - `test:` for adding tests
  - `ci:` for CI/CD pipeline changes
  - `build:` for build system or dependency changes
- ALWAYS use bun to run one-off JavaScript scripts.
