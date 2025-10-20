# @drexa/cli

Admin CLI tool for managing Drexa resources.

## Usage

From the project root:

```bash
bun drexa <command> [subcommand] [options]
```

## Commands

### `generate apikey`

Generate a new API key for authentication.

```bash
bun drexa generate apikey
```

The command will interactively prompt you for (using Node.js readline):
- **Prefix**: A short identifier for the key (e.g., 'proxy', 'admin'). Cannot contain dashes.
- **Key byte length**: Length of the key in bytes (default: 32)
- **Description**: A description of what this key is for
- **Expiration date**: Optional expiration date in YYYY-MM-DD format

The command will output:
- **Unhashed key**: Save this securely - it won't be shown again
- **Hashed key**: Store this in your database
- **Description**: The description you provided
- **Expiration date**: When the key expires (if set)

## Development

Run the CLI directly:

```bash
bun run apps/cli/index.ts <command>
```

## Project Structure

```
apps/cli/
├── index.ts                    # Main entry point
├── prompts.ts                  # Interactive prompt utilities
└── commands/                   # Command structure mirrors CLI structure
    └── generate/
        ├── index.ts           # Generate command group
        └── apikey.ts          # API key generation command
```

## Adding New Commands

1. Create a new directory under `commands/` for command groups
2. Create command files following the pattern in `commands/generate/apikey.ts`
3. Export commands from an `index.ts` in the command group directory
4. Register the command group in the main `index.ts`
