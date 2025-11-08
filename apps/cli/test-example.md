# Testing the CLI

To test the API key generation interactively, run:

```bash
bun drexa generate apikey
```

## Example Session

The CLI now uses **chalk** for beautiful colored output!

```
$ bun drexa generate apikey

🔑 Generate API Key

Enter API key prefix (e.g., 'proxy', 'admin'): testkey
Enter key byte length: (default: 32) 
Enter description: Test API Key for development
Enter expiration date (optional, format: YYYY-MM-DD): 

⏳ Generating API key...

✓ API Key Generated Successfully!

────────────────────────────────────────────────────────────

⚠️  IMPORTANT: Save the unhashed key now. It won't be shown again!

Unhashed Key (save this):
  sk-testkey-AbCdEfGhIjKlMnOpQrStUvWxYz0123456789

────────────────────────────────────────────────────────────

Hashed Key (store this in your database):
  $argon2id$v=19$m=4,t=3,p=1$...

Description:
  Test API Key for development

Expires At:
  Never

────────────────────────────────────────────────────────────
```

### Color Scheme
- **Prompts**: Cyan text with dimmed hints
- **Success messages**: Green with checkmark
- **Warnings**: Yellow with warning icon
- **Errors**: Red with X mark
- **Important data**: Green (unhashed key), dimmed (hashed key)
- **Separators**: Gray lines

## Testing with Invalid Input

### Invalid prefix (contains dash)
```bash
$ bun drexa generate apikey
Enter API key prefix (e.g., 'proxy', 'admin'): test-key
✗ Invalid prefix: cannot contain "-" character. Please use alphanumeric characters only.
```

### Invalid key byte length
```bash
$ bun drexa generate apikey
Enter API key prefix (e.g., 'proxy', 'admin'): testkey
Enter key byte length: (default: 32) -5
✗ Please enter a valid positive number
```

### Invalid date format
```bash
$ bun drexa generate apikey
Enter API key prefix (e.g., 'proxy', 'admin'): testkey
Enter key byte length: (default: 32) 
Enter description: Test
Enter expiration date (optional, format: YYYY-MM-DD): invalid-date
✗ Invalid date format. Please use YYYY-MM-DD
```

All error messages are displayed in red for better visibility.
