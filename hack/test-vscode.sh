#!/bin/sh -e
mkdir -p tmp/bin tmp/package test-project/.vscode
go build -o tmp/bin/gdshader-language-server .
mkdir -p test-project
version=0.0.$(date +%s)
(
	cd vscode-extension
	npm run package -- "$version" --out ../tmp/package
	npm version 0.0.0
)
code --install-extension tmp/package/gdshader-language-server-"$version".vsix --force
cat <<EOF >test-project/.vscode/settings.json
{
  "gdshader.trace.server": "verbose",
  "gdshader.danger.disableSafetyCheck": true,
  "gdshader.danger.serverPathOverride": "\${workspaceFolder}/../tmp/bin/gdshader-language-server",
}
EOF
code --disable-extensions test-project
