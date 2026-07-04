# Goshot

<a href="https://pkg.go.dev/github.com/watzon/goshot"><img src="https://pkg.go.dev/badge/github.com/watzon/goshot.svg" alt="Go Reference"></a>
<a href="https://goreportcard.com/report/github.com/watzon/goshot"><img src="https://goreportcard.com/badge/github.com/watzon/goshot" alt="Go Report Card"></a>
<a href="LICENSE"><img src="https://img.shields.io/github/license/watzon/goshot" alt="License"></a>

Goshot is a Go library and CLI for creating beautiful screenshots of code and terminal output, with syntax highlighting, window chrome, and rich backgrounds. Similar to [Carbon](https://carbon.now.sh) and [Silicon](https://github.com/Aloxaf/Silicon).

<div align="center">
    <img src=".github/example.png">
</div>

## ✨ Features

- 🎨 Syntax highlighting with hundreds of themes (chroma)
- 🖥 Terminal output rendering with full ANSI color support
- 🖼 Window chrome styles: macOS, Windows 11, GNOME, KDE Breeze
- 🌈 Backgrounds: solid colors, seven gradient types, and images
- 🕶 Automatic redaction of secrets (API keys, tokens, passwords)
- 💧 Drop shadows, rounded corners, blur effects
- 🔤 Embedded fonts plus system font discovery
- 💾 PNG, JPEG, and BMP export; clipboard and stdout output
- 🚀 Run a command and screenshot its output in one step

## Installation

```bash
# CLI
go install github.com/watzon/goshot/cmd/goshot@latest

# Library
go get github.com/watzon/goshot
```

Packages are also available for [Arch (AUR)](https://aur.archlinux.org/packages/goshot-bin) and [Ubuntu (PPA)](https://launchpad.net/~watzon/+archive/ubuntu/goshot):

```bash
yay -S goshot-bin                          # Arch
sudo add-apt-repository ppa:watzon/goshot  # Ubuntu
sudo apt install goshot
```

[![Packaging status](https://repology.org/badge/vertical-allrepos/goshot.svg)](https://repology.org/project/goshot/versions)

## CLI

```bash
# Screenshot a file
goshot main.go -o main.png

# Read from stdin, write to the clipboard
cat main.go | goshot -c

# Pick a theme, chrome, and background
goshot main.go -t catppuccin-mocha -C gnome -b '#1e1e2e'

# Gradient background with highlighted lines
goshot main.go --gradient-type linear \
    --gradient-stops '#4158D0;0' --gradient-stops '#C850C0;100' \
    --highlight-lines 10..14

# Redact secrets before sharing
goshot config.go --redact --redact-style blur

# Run a command and screenshot its output
goshot exec -A -p -- go test ./...
```

Run `goshot --help` for the full flag list, and `goshot themes`, `goshot fonts`, or `goshot languages` to see what's available.

### Configuration

Defaults for any flag can be set in `~/.config/goshot/config.yaml` as a flat map of flag names to values. Flags given on the command line always win.

```yaml
theme: catppuccin-mocha
chrome: mac
background: "#1e1e2e"
corner-radius: 12
```

## Library

The library is a small pipeline: **content** (code or terminal output) is rendered to an image, wrapped in **chrome**, and placed on a **background**.

```go
package main

import (
    "image/color"
    "log"

    "github.com/watzon/goshot"
    "github.com/watzon/goshot/background"
    "github.com/watzon/goshot/chrome"
    "github.com/watzon/goshot/code"
)

func main() {
    err := goshot.New().
        WithContent(code.New(`fmt.Println("Hello, goshot!")`).
            WithLanguage("go").
            WithTheme("dracula")).
        WithChrome(chrome.Mac().WithTitle("hello.go").Dark()).
        WithBackground(background.Gradient(background.LinearGradient,
            background.Stop{Color: color.RGBA{26, 27, 38, 255}, Position: 0},
            background.Stop{Color: color.RGBA{40, 42, 54, 255}, Position: 1},
        ).
            WithAngle(45).
            WithPadding(60).
            WithCornerRadius(10).
            WithShadow(background.NewShadow().WithBlur(20).WithOffset(0, 8))).
        Save("hello.png")
    if err != nil {
        log.Fatal(err)
    }
}
```

Terminal output works the same way with the `term` package:

```go
content := term.New(ansiOutput).
    WithTheme("catppuccin-mocha").
    WithAutoSize()

img, err := goshot.New().
    WithContent(content).
    WithChrome(chrome.Mac().Dark()).
    Image()
```

See [`examples/`](examples/) for runnable programs covering gradients, terminal rendering, and redaction.

### Packages

| Package      | Purpose                                              |
| ------------ | ---------------------------------------------------- |
| `goshot`     | The canvas pipeline and image export                 |
| `code`       | Syntax-highlighted code rendering                    |
| `term`       | Terminal output rendering (ANSI escapes, 256 colors) |
| `chrome`     | Window decorations (mac, windows, gnome, breeze)     |
| `background` | Solid, gradient, and image backdrops                 |
| `fonts`      | Embedded and system font loading                     |

## License

This project is licensed under the MIT License — see the [LICENSE](LICENSE) file for details.
