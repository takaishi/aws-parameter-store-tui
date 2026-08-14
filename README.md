# aws-parameter-store-tui

A TUI for browsing AWS Systems Manager Parameter Store: search parameters, view values, and copy secrets to the clipboard. The command name is `pstui`.

## Install

Homebrew:

```bash
brew install --cask takaishi/tap/pstui
```

Go:

```bash
go install github.com/takaishi/aws-parameter-store-tui/cmd/pstui@latest
```

Prebuilt binaries are also available on the [releases page](https://github.com/takaishi/aws-parameter-store-tui/releases).

## Usage

```bash
pstui [--profile <profile>] [--region <region>]
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

## Required IAM permissions

- `ssm:DescribeParameters`
- `ssm:GetParameter`
- `kms:Decrypt` (for SecureString parameters)

## License

[MIT](./LICENSE). Licenses of all dependencies are bundled in [CREDITS](./CREDITS); regenerate it with `make credits`, and verify dependency licenses with `make license-check`.
