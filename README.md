# wand

**`wand`** is a tiny, cross-platform command runner driven by a simple **YAML config file**, written
in **Go**. Define your commands and subcommands in a `wand.yml`, and run them from anywhere in your
project tree.

![Release](https://img.shields.io/github/v/release/chenasraf/wand)
![Downloads](https://img.shields.io/github/downloads/chenasraf/wand/total)
![License](https://img.shields.io/github/license/chenasraf/wand)

---

## 🚀 Features

- **Simple YAML config**: define commands, descriptions, and nested subcommands in a single file.
- **Auto-discovery**: finds `wand.yml` by searching the current directory, parent directories, `~/`,
  and `~/.config/`.
- **Nested subcommands**: commands can have arbitrarily deep children.
- **Positional arguments**: pass arguments to commands and reference them with `$1`, `$2`, `$@`.
- **Custom flags**: define typed flags (string or bool) with aliases, defaults, and descriptions,
  accessible as `$WAND_FLAG_<NAME>` environment variables.
- **Global flags**: declare flags once in `.config` and use them with any command, before or after
  the command name.
- **Custom binary name**: rename the tool in help output to present a config as its own CLI.
- **Environment variables**: define env vars globally in `.config` or per command, with
  command-level overrides.
- **Working directory**: override the working directory for any command.
- **Aliases**: define alternate names for commands.
- **Confirmation prompts**: require `y/N` confirmation before running destructive commands.
- **Pre/post hooks**: chain other wand commands to run before or after a command, with full
  flag/argument forwarding.
- **Built-in help**: auto-generated `--help` for every command and subcommand.
- **Shell execution**: runs commands via your `$SHELL` with proper stdin/stdout/stderr passthrough.

---

## 🎯 Installation

### Download Precompiled Binaries

Grab the latest release for **Linux**, **macOS**, or **Windows**:

- [Releases →](https://github.com/chenasraf/wand/releases/latest)

### Homebrew (macOS/Linux)

Install directly from the tap:

```bash
brew install chenasraf/tap/wand
```

Or tap and then install the package:

```bash
brew tap chenasraf/tap
brew install wand
```

### From Source

```bash
git clone https://github.com/chenasraf/wand
cd wand
make build
```

---

## ✨ Getting Started

Create a `wand.yml` in your project root:

```yaml
main:
  description: run the main command
  cmd: echo hello from wand

build:
  description: build the project
  cmd: go build -o myapp

test:
  description: run tests
  cmd: go test -v ./...
  children:
    coverage:
      description: run tests with coverage
      cmd: go test -coverprofile=coverage.out ./...
```

### Run a command

```bash
# run the main (default) command
wand

# run a named command
wand build

# run a nested subcommand
wand test coverage

# show help
wand --help
wand test --help
```

---

## 📁 Config Resolution

`wand` searches for `wand.yml` (or `wand.yaml`) in the following order:

1. Current working directory (`./wand.yml`)
2. Parent directories (searching upward to the filesystem root)
3. Home directory (`~/.wand.yml`)
4. Config directory (`~/.config/wand.yml`)

The first config file found is used.

You can override config discovery with an explicit path:

```bash
# via flag
wand --wand-file ./other-config.yml build

# via environment variable
WAND_FILE=./other-config.yml wand build
```

The `--wand-file` flag takes precedence over `WAND_FILE`.

---

## 📖 Config Reference

Each top-level key defines a command, except `.config`, which holds settings that apply to the whole
file. The special key `main` becomes the root (no-argument) command.

### `.config` fields

| Field      | Type                | Description                                   |
| ---------- | ------------------- | --------------------------------------------- |
| `shell`    | `string` or `map`   | Shell used to run commands, optionally per OS |
| `env`      | `map[string]string` | Environment variables for every command       |
| `flags`    | `map[string]Flag`   | Flags available to every command (see below)  |
| `bin_name` | `string`            | Name shown in help output (default `wand`)    |

### Command fields

| Field             | Type                 | Description                                 |
| ----------------- | -------------------- | ------------------------------------------- |
| `description`     | `string`             | Short description shown in `--help`         |
| `cmd`             | `string`             | Shell command to execute                    |
| `children`        | `map[string]Command` | Nested subcommands (same structure)         |
| `flags`           | `map[string]Flag`    | Custom flags (see below)                    |
| `env`             | `map[string]string`  | Environment variables for this command      |
| `working_dir`     | `string`             | Working directory for the command           |
| `aliases`         | `[]string`           | Alternate names for the command             |
| `confirm`         | `bool` or `string`   | Prompt for confirmation before running      |
| `confirm_default` | `string`             | Default answer: `"yes"` or `"no"` (default) |
| `pre`             | `[]string`           | Wand commands to run before `cmd`           |
| `post`            | `[]string`           | Wand commands to run after `cmd`            |

### Flag fields

| Field         | Type     | Description                                       |
| ------------- | -------- | ------------------------------------------------- |
| `alias`       | `string` | Single-letter shorthand (e.g. `o` for `-o`)       |
| `description` | `string` | Description shown in `--help`                     |
| `default`     | `any`    | Default value (`string` or `bool`)                |
| `type`        | `string` | `"bool"` for boolean flags, omit for string flags |

---

## 📌 Positional Arguments

Commands receive any extra arguments passed on the command line. Use `$1`, `$2`, etc. for specific
positions, or `$@` for all arguments:

```yaml
greet:
  description: greet someone
  cmd: echo "Hello, $1! You said: $@"
```

```bash
wand greet world foo bar
# → Hello, world! You said: world foo bar
```

---

## 🚩 Flags

Define custom flags per command. Flag values are exposed as `$WAND_FLAG_<NAME>` environment
variables (uppercased):

```yaml
build:
  description: build the project
  cmd: |
    echo "output=$WAND_FLAG_OUTPUT verbose=$WAND_FLAG_VERBOSE"
  flags:
    output:
      alias: o
      description: output path
      default: ./bin
    verbose:
      alias: v
      description: enable verbose output
      type: bool
```

```bash
wand build --output ./dist --verbose
# → output=./dist verbose=true

wand build -o ./dist -v
# → output=./dist verbose=true

wand build
# → output=./bin verbose=false
```

### Global flags

Flags declared under `.config` are available to every command, and may be passed either before or
after the command name:

```yaml
.config:
  flags:
    profile:
      alias: p
      description: config profile
      default: dev
    verbose:
      alias: V
      description: enable verbose output
      type: bool

build:
  cmd: echo "profile=$WAND_FLAG_PROFILE verbose=$WAND_FLAG_VERBOSE"
  children:
    docs:
      cmd: echo "docs profile=$WAND_FLAG_PROFILE"
```

```bash
wand build --profile prod
wand --profile prod build
wand -p prod build docs
# → profile=prod verbose=false
```

A command flag of the same name shadows the global one for that command:

```yaml
.config:
  flags:
    profile:
      default: dev

deploy:
  cmd: echo $WAND_FLAG_PROFILE
  flags:
    profile:
      default: staging
```

```bash
wand deploy
# → staging
```

Global flag names and aliases must not collide with each other, with a command's own flags, or with
wand's built-in `--wand-file` and `--help`.

---

## 🏷️ Binary Name

Set `bin_name` to have help and usage output name your tool instead of `wand`. This suits a config
exposed through a wrapper or alias, so it reads as its own CLI:

```yaml
.config:
  bin_name: nxc

sync:
  description: sync files
  cmd: ./sync.sh
```

```bash
wand --wand-file ~/.config/wand/nextcloud.yml --help
```

```
Usage:
  nxc [command]

Available Commands:
  sync        sync files

Use "nxc [command] --help" for more information about a command.
```

The name flows through nested command paths (`nxc sync --help`) and generated completion scripts.
`--wand-file` drops out of the help output when `bin_name` is set, since the renamed tool presents
itself as its own CLI — the flag keeps working, so a wrapper can still point at the config:

```bash
#!/bin/sh
exec wand --wand-file ~/.config/wand/nextcloud.yml "$@"
```

---

## 🌍 Environment Variables

Define environment variables globally in `.config` or per command. Command-level env vars override
global ones:

```yaml
.config:
  env:
    NODE_ENV: production

build:
  description: build the project
  cmd: echo "env=$NODE_ENV out=$OUTPUT_DIR"
  env:
    OUTPUT_DIR: ./dist
```

```bash
wand build
# → env=production out=./dist
```

---

## ⚠️ Confirmation Prompts

Add `confirm: true` for a default prompt, or provide a custom message:

```yaml
deploy:
  description: deploy to production
  cmd: ./deploy.sh
  confirm: 'Deploy to production?'

clean:
  description: remove all build artifacts
  cmd: rm -rf dist/
  confirm: true

restart:
  description: restart service
  cmd: systemctl restart myapp
  confirm: 'Restart the service?'
  confirm_default: 'yes'
```

```bash
wand deploy
# → Deploy to production? [y/N]
```

---

## 🔗 Pre & Post Hooks

Use `pre` and `post` to run other wand commands before or after a command. Each entry is a
shell-style string: the first token is the wand command name (subcommands are nested with
spaces), followed by any args and flags.

```yaml
lint:
  description: lint the project
  cmd: golangci-lint run

test:
  description: run tests
  cmd: go test ./...

build:
  description: build the project
  pre:
    - lint
    - test
  post:
    - 'echo "build done: $WAND_FLAG_OUTPUT"'
  flags:
    output:
      alias: o
      default: ./bin
  cmd: go build -o $WAND_FLAG_OUTPUT
```

```bash
wand build -o ./dist
# runs: lint → test → go build -o ./dist → echo "build done: ./dist"
```

### Forwarding flags

Entries are passed through environment variable expansion (`$VAR`, `${VAR}`) before being
parsed, so `$WAND_FLAG_<NAME>` references resolve to the current command's flag values (global flags
included):

```yaml
deploy:
  flags:
    target:
      description: deploy target
      default: staging
  pre:
    - 'notify --channel deploys --message "deploying to $WAND_FLAG_TARGET"'
  cmd: ./deploy.sh $WAND_FLAG_TARGET
```

Arbitrary flags and arguments can be passed directly:

```yaml
release:
  pre:
    - 'test --verbose'
    - 'build --output ./dist'
  cmd: ./release.sh
```

### Failure semantics

- If a `pre` entry fails, the main `cmd` and remaining `pre`/`post` entries are skipped.
- If the main `cmd` fails, no `post` entries run.
- If a `post` entry fails, subsequent `post` entries are skipped.

A command may omit `cmd` and define only `pre`/`post` to act as a pure aggregator.

### Private commands

Prefix a command name with `_` to mark it as private: it is hidden from `--help` output but
remains fully runnable, both directly (useful for testing) and from `pre`/`post` entries. The
same rule applies to nested children.

```yaml
build:
  pre:
    - _ensure-deps
  cmd: go build

_ensure-deps:
  description: install required tools
  cmd: ./scripts/install-deps.sh
```

```bash
wand --help        # _ensure-deps is not listed
wand _ensure-deps  # still runs directly
wand build         # runs _ensure-deps then the build
```

---

## 🛠️ Contributing

I am developing this package on my free time, so any support, whether code, issues, or just stars is
very helpful to sustaining its life. If you are feeling incredibly generous and would like to donate
just a small amount to help sustain this project, I would be very very thankful!

<a href='https://ko-fi.com/casraf' target='_blank'>
<img height='36' style='border:0px;height:36px;' src='https://cdn.ko-fi.com/cdn/kofi1.png?v=3' alt='Buy Me a Coffee at ko-fi.com' />
</a>

I welcome any issues or pull requests on GitHub. If you find a bug, or would like a new feature,
don't hesitate to open an appropriate issue and I will do my best to reply promptly.

---

## 📜 License

`wand` is licensed under the [CC0-1.0 License](/LICENSE).
