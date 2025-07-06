// Copyright (c) 2025 Adam Snyder <https://armsnyder.com> and contributors
// SPDX-License-Identifier: MIT

package ast_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/alecthomas/participle/v2/lexer"
	"github.com/armsnyder/gdshader-language-server/internal/ast"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/samber/lo"

	. "github.com/onsi/gomega"
)

func TestParser(t *testing.T) {
	g := NewWithT(t)
	content := lo.Must(os.ReadFile("testdata/valid/shader.gdshader"))
	const filename = "shader.gdshader"
	shader, err := ast.Parse(filename, bytes.NewReader(content))
	g.Expect(err).ToNot(HaveOccurred())
	expected := &ast.File{
		Declarations: []*ast.Declaration{
			{
				UniformDecl: &ast.UniformDecl{
					Type: "sampler2D",
					Name: "texture",
				},
			},
			{
				FunctionDecl: &ast.FunctionDecl{
					ReturnType: "void",
					Name:       "vertex",
					Body:       &ast.BlockStmt{},
				},
			},
			{
				FunctionDecl: &ast.FunctionDecl{
					ReturnType: "void",
					Name:       "fragment",
					Body:       &ast.BlockStmt{},
				},
			},
		},
	}

	g.Expect(shader).To(BeComparableTo(expected, IgnorePos))
}

func TestCanParseAllValidPrograms(t *testing.T) {
	files := lo.Must(os.ReadDir("testdata/valid"))
	for _, file := range files {
		t.Run(file.Name(), func(t *testing.T) {
			content := lo.Must(os.ReadFile(filepath.Join("testdata/valid", file.Name())))
			_, err := ast.Parse(file.Name(), bytes.NewReader(content))
			NewWithT(t).Expect(err).ToNot(HaveOccurred())
		})
	}
}

var IgnorePos = cmpopts.IgnoreTypes(lexer.Position{})
