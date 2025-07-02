// MIT License
//
// Copyright (c) 2025 Adam Snyder
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

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
