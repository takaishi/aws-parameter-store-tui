# aws-tui

A collection of TUI tools for AWS services. Each tool is distributed as its own binary from this repository.

| Tool | Description |
| --- | --- |
| [aws-parameter-store-tui](#aws-parameter-store-tui) | Browse AWS Systems Manager Parameter Store |
| [aws-secrets-manager-tui](#aws-secrets-manager-tui) | Browse AWS Secrets Manager |
| [aws-ecs-tui](#aws-ecs-tui) | Browse Amazon ECS clusters, services, and tasks |

## aws-parameter-store-tui

A TUI for browsing AWS Systems Manager Parameter Store: search parameters, view values, and copy secrets to the clipboard.

### Install

Homebrew:

```bash
brew install --cask takaishi/tap/aws-parameter-store-tui
```

Go:

```bash
go install github.com/takaishi/aws-tui/cmd/aws-parameter-store-tui@latest
```

Prebuilt binaries are also available on the [releases page](https://github.com/takaishi/aws-tui/releases).

### Usage

```bash
aws-parameter-store-tui [--profile <profile>] [--region <region>]
```

AWS credentials are resolved via the standard AWS SDK chain (`AWS_PROFILE`, shared config, SSO, etc.).

#### Key bindings

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
| `t` | toggle key/value view ↔ raw view (values that are a JSON object) |
| `↑` / `↓` | scroll value |
| `esc` / `q` | back to list |

SecureString values are decrypted via `GetParameter` with `WithDecryption` and shown masked by default.

Values that are a JSON object (e.g. `{"username":"...","password":"..."}`) are shown as a key/value list by default; press `t` to switch to the raw JSON. Copy always copies the raw value.

### Required IAM permissions

- `ssm:DescribeParameters`
- `ssm:GetParameter`
- `kms:Decrypt` (for SecureString parameters)

## aws-secrets-manager-tui

A TUI for browsing AWS Secrets Manager: search secrets, view values, and copy them to the clipboard.

### Install

Homebrew:

```bash
brew install --cask takaishi/tap/aws-secrets-manager-tui
```

Go:

```bash
go install github.com/takaishi/aws-tui/cmd/aws-secrets-manager-tui@latest
```

Prebuilt binaries are also available on the [releases page](https://github.com/takaishi/aws-tui/releases).

### Usage

```bash
aws-secrets-manager-tui [--profile <profile>] [--region <region>]
```

Key bindings are the same as aws-parameter-store-tui. Secret values are always shown masked by default; press `s` in the detail view to reveal. Binary secrets are displayed base64-encoded.

### Required IAM permissions

- `secretsmanager:ListSecrets`
- `secretsmanager:GetSecretValue`
- `kms:Decrypt` (for secrets encrypted with a customer managed key)

## aws-ecs-tui

A read-only TUI for browsing Amazon ECS: drill down from clusters to services to tasks, check deployment rollout state and service events, inspect why tasks stopped, and view task definitions including environment variables.

### Install

Homebrew:

```bash
brew install --cask takaishi/tap/aws-ecs-tui
```

Go:

```bash
go install github.com/takaishi/aws-tui/cmd/aws-ecs-tui@latest
```

Prebuilt binaries are also available on the [releases page](https://github.com/takaishi/aws-tui/releases).

### Usage

```bash
aws-ecs-tui [--profile <profile>] [--region <region>] [--cluster <cluster>]
```

With `--cluster`, the TUI starts directly at that cluster's service list.

The UI is a three-pane column layout:

```
┌ clusters ──────┬ services ──────┬ service ────────────────────────────┐
│ > prod         │ > web          │   Detail & events                   │
│   staging      │   worker       │   Task definition (my-app:42)      │
│                │                │ > 1a2b3c… (RUNNING (HEALTHY))       │
│                │                │   4d5e6f… (STOPPED — Essential …)   │
└────────────────┴────────────────┴─────────────────────────────────────┘
```

The rightmost pane lists the service's running and recently stopped tasks (stopped reason inline), preceded by entries for the service's detail & events and its task definition (images, env vars, secrets, log config). `enter` on a task (or on Detail & events / Task definition) opens its detail — containers, exit codes, private IP, stop code — in the rightmost pane, keeping the column layout; ancestor panes stay visible on the left.

Key bindings: type to fuzzy-filter the focused pane, `↑`/`↓` to move, `←`/`→` (or `tab`/`shift+tab`) to switch panes, `enter` to open, `ctrl+y` to copy the selected item's ARN, `ctrl+r` to reload the focused pane, `esc` to focus the previous pane (quit at the leftmost). The pane to the right of the focused one previews the current selection automatically, debounced so that holding `↓` doesn't fire an API call per row.

### Required IAM permissions

- `ecs:ListClusters` / `ecs:DescribeClusters`
- `ecs:ListServices` / `ecs:DescribeServices`
- `ecs:ListTasks` / `ecs:DescribeTasks`
- `ecs:DescribeTaskDefinition`

## License

[MIT](./LICENSE). Licenses of all dependencies are bundled in [CREDITS](./CREDITS); regenerate it with `make credits`, and verify dependency licenses with `make license-check`.
