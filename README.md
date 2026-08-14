# aws-tui

A collection of TUI tools for AWS services. Each tool is distributed as its own binary from this repository.

| Tool | Description |
| --- | --- |
| [aws-parameter-store-tui](#aws-parameter-store-tui) | Browse AWS Systems Manager Parameter Store |
| [aws-secrets-manager-tui](#aws-secrets-manager-tui) | Browse AWS Secrets Manager |
| [aws-ecs-tui](#aws-ecs-tui) | Browse Amazon ECS clusters, services, and tasks |
| [aws-security-group-tui](#aws-security-group-tui) | Browse Amazon EC2 security groups |
| [aws-ec2-tui](#aws-ec2-tui) | Browse Amazon EC2 instances |
| [aws-route53-tui](#aws-route53-tui) | Browse Amazon Route 53 hosted zones and records |
| [aws-cloudwatch-logs-tui](#aws-cloudwatch-logs-tui) | Browse Amazon CloudWatch Logs groups and streams |

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

## aws-security-group-tui

A read-only TUI for browsing Amazon EC2 security groups: search groups, and view inbound/outbound rules including CIDR blocks, referenced security groups, and prefix lists.

### Install

Homebrew:

```bash
brew install --cask takaishi/tap/aws-security-group-tui
```

Go:

```bash
go install github.com/takaishi/aws-tui/cmd/aws-security-group-tui@latest
```

Prebuilt binaries are also available on the [releases page](https://github.com/takaishi/aws-tui/releases).

### Usage

```bash
aws-security-group-tui [--profile <profile>] [--region <region>]
```

Key bindings are the same as aws-parameter-store-tui. `enter` on a security group shows its inbound and outbound rules; `ctrl+y` copies the group ID.

### Required IAM permissions

- `ec2:DescribeSecurityGroups`

## aws-ec2-tui

A read-only TUI for browsing Amazon EC2 instances: search instances, and view instance type, state, IPs, security groups, and tags.

### Install

Homebrew:

```bash
brew install --cask takaishi/tap/aws-ec2-tui
```

Go:

```bash
go install github.com/takaishi/aws-tui/cmd/aws-ec2-tui@latest
```

Prebuilt binaries are also available on the [releases page](https://github.com/takaishi/aws-tui/releases).

### Usage

```bash
aws-ec2-tui [--profile <profile>] [--region <region>]
```

Key bindings are the same as aws-parameter-store-tui. `enter` on an instance shows its tags; `ctrl+y` copies the instance ID.

### Required IAM permissions

- `ec2:DescribeInstances`

## aws-route53-tui

A read-only TUI for browsing Amazon Route 53: drill down from hosted zones to their record sets, and view record values (including alias targets).

### Install

Homebrew:

```bash
brew install --cask takaishi/tap/aws-route53-tui
```

Go:

```bash
go install github.com/takaishi/aws-tui/cmd/aws-route53-tui@latest
```

Prebuilt binaries are also available on the [releases page](https://github.com/takaishi/aws-tui/releases).

### Usage

```bash
aws-route53-tui [--profile <profile>] [--region <region>]
```

The UI is a two-pane column layout: hosted zones on the left, that zone's record sets on the right. `enter` on a record shows its type, TTL, routing policy fields, and values (or alias target). Key bindings: type to fuzzy-filter the focused pane, `↑`/`↓` to move, `←`/`→` (or `tab`/`shift+tab`) to switch panes, `ctrl+y` to copy the selected hosted zone ID, `ctrl+r` to reload the focused pane, `esc` to focus the previous pane (quit at the leftmost).

### Required IAM permissions

- `route53:ListHostedZones`
- `route53:ListResourceRecordSets`

## aws-cloudwatch-logs-tui

A read-only TUI for browsing Amazon CloudWatch Logs: drill down from log groups to their log streams, and tail the most recent events in a stream.

### Install

Homebrew:

```bash
brew install --cask takaishi/tap/aws-cloudwatch-logs-tui
```

Go:

```bash
go install github.com/takaishi/aws-tui/cmd/aws-cloudwatch-logs-tui@latest
```

Prebuilt binaries are also available on the [releases page](https://github.com/takaishi/aws-tui/releases).

### Usage

```bash
aws-cloudwatch-logs-tui [--profile <profile>] [--region <region>]
```

The UI is a two-pane column layout: log groups on the left, that group's log streams (most recently active first) on the right. `enter` on a stream shows its most recent events (up to 200). Key bindings: type to fuzzy-filter the focused pane, `↑`/`↓` to move, `←`/`→` (or `tab`/`shift+tab`) to switch panes, `ctrl+y` to copy the selected item's ARN, `ctrl+r` to reload the focused pane, `esc` to focus the previous pane (quit at the leftmost).

### Required IAM permissions

- `logs:DescribeLogGroups`
- `logs:DescribeLogStreams`
- `logs:GetLogEvents`

## License

[MIT](./LICENSE). Licenses of all dependencies are bundled in [CREDITS](./CREDITS); regenerate it with `make credits`, and verify dependency licenses with `make license-check`.
