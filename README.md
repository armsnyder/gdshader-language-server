# Godot Shader Language Server

![Go](https://img.shields.io/badge/Made%20with-Go-00ADD8?logo=go&logoColor=white)
![License](https://img.shields.io/github/license/armsnyder/gdshader-language-server)

External editor support for `.gdshader` files.

> [!WARNING]
> 🚧 Early Work in Progress
>
> This project is in its infancy and currently has no features. Feel free to ⭐
> the repo to track progress and signal to me that there is interest!

Godot's shader language is powerful, but editing `.gdshader` files outside the
Godot editor is painful. This project aims to bring proper language tooling
(autocomplete, hover, references, etc.) to editors like Neovim and VSCode.

## 📦 Install

Install from source:

```shell
go install github.com/armsnyder/gdshader-language-server@latest
```

## ⚙️  Configure

### Neovim

Add the following to your `init.lua`, adjusting the path to the
`gdshader-language-server` binary if necessary:

```lua
vim.api.nvim_create_autocmd("FileType", {
  pattern = "gdshader",
  callback = function()
    vim.lsp.start({
      name = "gdshader",
      cmd = { vim.fs.expand('$HOME/go/bin/gdshader-language-server') },
    })
  end,
})
```

### VSCode

Coming soon? Contributions welcome!

## Roadmap

Planned features:

- [ ] Go to definition
- [ ] Find references
- [ ] Formatting
- [ ] Hover (show documentation)
- [ ] Signature help
- [ ] VSCode wrapper extension

## 🤝 Contributing

I love to see issues and pull requests! Just note that this is a side project
for me, and I cannot promise to respond quickly. I will generally accept pull
requests which are relevant to the project goals, are tested, and follow
existing code conventions.

### 📁 Code structure

```graphql
.
├── main.go # Entry point
└── internal
    ├── ast     # .gdshader file parser library (application agnostic)
    ├── handler # Main application logic
    └── lsp     # LSP server library (application agnostic)
```

## License

MIT
