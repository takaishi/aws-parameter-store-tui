# sstui

A TUI for browsing AWS Systems Manager Parameter Store: search parameters, view values, and copy secrets to the clipboard.

## Install

```bash
go install github.com/takaishi/aws-ss-tui/cmd/sstui@latest
```

## Usage

```bash
sstui [--profile <profile>] [--region <region>]
```

AWS credentials are resolved via the standard AWS SDK chain (`AWS_PROFILE`, shared config, SSO, etc.).

### Key bindings

List view:

| Key | Action |
| --- | --- |
| type | fuzzy-filter parameters (fzf-style, ranked by match score) |
| `↑` / `↓` (`ctrl+p` / `ctrl+n`) | move cursor |
| `enter` | view parameter detail |
| `ctrl+y` | copy the selected parameter's value to clipboard |
| `ctrl+r` | reload parameters |
| `esc` / `ctrl+c` | quit |

Detail view:

| Key | Action |
| --- | --- |
| `y` / `c` | copy value to clipboard |
| `s` | reveal / mask value (SecureString) |
| `↑` / `↓` | scroll value |
| `esc` / `q` | back to list |

SecureString values are decrypted via `GetParameter` with `WithDecryption` and shown masked by default.

## License notices

Licenses of all dependencies are bundled in [CREDITS](./CREDITS). Regenerate it with `make credits`.

## Required IAM permissions

- `ssm:DescribeParameters`
- `ssm:GetParameter`
- `kms:Decrypt` (for SecureString parameters)
