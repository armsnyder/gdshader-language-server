# Godot Shader Language Server

[![GitHub release](https://img.shields.io/github/v/release/armsnyder/gdshader-language-server)](https://github.com/armsnyder/gdshader-language-server/releases/latest)
[![CI](https://github.com/armsnyder/gdshader-language-server/actions/workflows/ci.yaml/badge.svg)](https://github.com/armsnyder/gdshader-language-server/actions/workflows/ci.yaml)
[![Coverage](https://img.shields.io/endpoint?url=https://gist.githubusercontent.com/armsnyder/6858b1591174caeee65c12bec018bbad/raw/coverage.json)](https://armsnyder.github.io/gdshader-language-server/cover.html)
[![Go Report Card](https://goreportcard.com/badge/github.com/armsnyder/gdshader-language-server)](https://goreportcard.com/report/github.com/armsnyder/gdshader-language-server)

External editor support for `.gdshader` files.

> [!WARNING]
> 🚧 Early Work in Progress
>
> This project is in its infancy and currently only supports some basic keyword
> completion. Feel free to ⭐ the repo to track progress and signal to me that
> there is interest!

Godot's shader language is powerful, but editing `.gdshader` files outside the
Godot editor is painful. This project aims to bring proper language tooling
(autocomplete, hover, references, etc.) to editors like Neovim and VSCode. It
aims to be memory-efficient and editor-agnostic.

## 🌱 Prior Work

1. [@GodOfAvacyn](https://github.com/GodOfAvacyn) is the author of the
   [treesitter plugin](https://github.com/GodOfAvacyn/tree-sitter-gdshader) and
   [language server](https://github.com/GodOfAvacyn/gdshader-lsp) for the Godot
   shader language. Their treesitter plugin is great! As of writing, their
   language server has many false positive diagnostics, and the project became
   inactive while they were [working on a full
   rewrite](https://github.com/GodOfAvacyn/gdshader-lsp/issues/3#issuecomment-2176364609).

2. [@AlfishSoftware](https://github.com/AlfishSoftware) is the author of the
   [Godot Files VSCode
   Extension](https://github.com/AlfishSoftware/godot-files-vscode), which
   includes some support for `.gdshader` files. However, it is VSCode-only. If
   you are a VSCode user, I recommend checking it out!

3. There is an [official Godot VSCode
   plugin](https://github.com/godotengine/godot-vscode-plugin), but it has
   minimal shader support and is VSCode-only.

## 📦 Install

### VSCode

[Install the extension](https://marketplace.visualstudio.com/items?itemName=armsnyder.gdshader-language-server)

### Neovim

#### 1. Choose an installation method

##### Homebrew

```shell
brew install armsnyder/tap/gdshader-language-server
```

##### Go

```shell
go install github.com/armsnyder/gdshader-language-server@latest
```

##### Github Releases

[Go to releases](https://github.com/armsnyder/gdshader-language-server/releases)

#### 2. Configure Neovim

Create a `~/.config/nvim/after/ftplugin/gdshader.lua` file with the
following content, assuming `gdshader-language-server` is in your `$PATH`:

```lua
vim.lsp.start({
  name = "gdshader",
  cmd = { 'gdshader-language-server' },
  capabilities = vim.lsp.protocol.make_client_capabilities(),
})
```

## Roadmap

Planned features:

- [x] Basic keyword completion
- [x] Basic shader-type-dependent global built-in completion
      (`VERTEX`, `NORMAL`, etc.)
- [x] VSCode wrapper extension
- [x] [Grammar](https://code.visualstudio.com/api/references/contribution-points#contributes.grammars)
      for the VSCode extension
- [ ] Make the code more maintainable by generating rules based on the official
      Godot documentation
- [ ] Built-ins for shader types other than `spatial`
- [ ] More advanced completion (functions, variables, etc.)
- [ ] Go to definition
- [ ] Find references
- [ ] Formatting
- [ ] Hover (show documentation)
- [ ] Signature help

## 🤝 Contributing

I love to see issues and pull requests! Just note that this is a side project
for me, and I cannot promise to respond quickly. I will generally accept pull
requests which are relevant to the project goals, are tested, and follow
existing code conventions.

### 📁 Code structure

```graphql
.
├── main.go  # Entry point
└── internal
    ├── app       # Main application logic
    ├── ast       # .gdshader file parser library (application agnostic)
    ├── lsp       # LSP server library (application agnostic)
    └── testutil  # Test utilities for all packages
```

## License

MIT
