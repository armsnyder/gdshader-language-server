# Godot Shader Language Server

![Go](https://img.shields.io/badge/Made%20with-Go-00ADD8?logo=go&logoColor=white)
![License](https://img.shields.io/github/license/armsnyder/gdshader-language-server)

External editor support for `.gdshader` files.

> [!WARNING]
> 🚧 Early Work in Progress
>
> This project is in its infancy and currently only supports some basic keyword
> completion. Feel free to ⭐ the repo to track progress and signal to me that
> there is interest!

Godot's shader language is powerful, but editing `.gdshader` files outside the
Godot editor is painful. This project aims to bring proper language tooling
(autocomplete, hover, references, etc.) to editors like Neovim and VSCode.

## 🌱 Prior Work

[@GodOfAvacyn](https://github.com/GodOfAvacyn) is the wonderful author of the
[treesitter plugin](https://github.com/GodOfAvacyn/tree-sitter-gdshader) and
[language server](https://github.com/GodOfAvacyn/gdshader-lsp) for the Godot
shader language. Their treesitter plugin is great! As of writing, their
language server has many false positive diagnostics, and the project became
inactive while they were
[working on a full rewrite](https://github.com/GodOfAvacyn/gdshader-lsp/issues/3#issuecomment-2176364609).
I decided to start this new project to fill the gap for myself.

## 📦 Install

### Neovim

1. Install by downloading the latest release or building from source:

   ```shell
   wget https://github.com/armsnyder/gdshader-language-server/releases/latest/download/gdshader-language-server_$(uname -s)_$(uname -m).tar.gz
   ```

   _or_

   ```shell
   go install github.com/armsnyder/gdshader-language-server@latest
   ```

1. Create a `~/.config/nvim/after/ftplugin/gdshader.lua` file with the following
   content, adjusting the path to the `gdshader-language-server` binary if
   necessary:

   ```lua
   vim.lsp.start({
     name = "gdshader",
     cmd = { vim.fs.expand('~/go/bin/gdshader-language-server') },
     capabilities = vim.lsp.protocol.make_client_capabilities(),
   })
   ```

### VSCode

Coming soon? Contributions welcome!

## Roadmap

Planned features:

- [x] Basic keyword completion
- [x] Basic shader-type-dependent global built-in completion
      (`VERTEX`, `NORMAL`, etc.)
- [ ] Built-ins for shader types other than `spatial`
- [ ] More advanced completion (functions, variables, etc.)
- [ ] Go to definition
- [ ] Find references
- [ ] Formatting
- [ ] Hover (show documentation)
- [ ] Signature help
- [ ] VSCode wrapper extension
- [ ] Make the code more maintainable by generating rules based on the official
      Godot documentation

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
