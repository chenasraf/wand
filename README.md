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

---

## 📖 Config Reference

Each top-level key defines a command. The special key `main` becomes the root (no-argument) command.

| Field         | Type                 | Description                         |
| ------------- | -------------------- | ----------------------------------- |
| `description` | `string`             | Short description shown in `--help` |
| `cmd`         | `string`             | Shell command to execute            |
| `children`    | `map[string]Command` | Nested subcommands (same structure) |
| `flags`       | `map[string]Flag`    | Custom flags (see below)            |

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
