# Project tech stack
frontend: react, shadcn, tailwindcss
backend: convex

# Project structure
This project uses npm workspaces.
- `packages/convex` - convex functions and models
- `apps/drive-web` - frontend dashboard
- `apps/file-proxy` - proxies uploaded files via opaque share tokens
- `packages/path` - path utils

# General Guidelines

- For frontend dashboard guidelines, refer to dev/docs/dashboard.md
- Always use bun for package management and running npm scripts.
- When commiting, follow conventional commits: `[type]: msg` and clarify in commit body if necessary. keep your commit header to <= 50 characters.
  - `feat:` for new features
  - `fix:` for bug fixes  
  - `docs:` for documentation
  - `refactor:` for code restructuring
  - `style:` for formatting changes
  - `test:` for adding tests
  - `ci:` for CI/CD pipeline changes
  - `build:` for build system or dependency changes

# Code styles

- file names should always be kebab-case except in convex code, in which case try to use single word file names.
