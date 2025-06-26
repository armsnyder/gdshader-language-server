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

package ast

import "github.com/alecthomas/participle/v2/lexer"

// File is the root of a parsed .gdshader file.
type File struct {
	Pos          lexer.Position
	Declarations []*Declaration `@@*`
}

// Declaration is a top-level declaration.
type Declaration struct {
	UniformDecl  *UniformDecl  `@@`
	FunctionDecl *FunctionDecl `| @@`
}

// UniformDecl is a uniform variable declaration.
type UniformDecl struct {
	Type string `"uniform" @Ident`
	Name string `@Ident ";"`
}

// FunctionDecl is a function declaration.
type FunctionDecl struct {
	ReturnType string     `@Ident`
	Name       string     `@Ident "(" ")"`
	Body       *BlockStmt `@@`
}

// BlockStmt is a block of statements enclosed in braces.
type BlockStmt struct {
	Stmts []*Stmt `"{" @@* "}"`
}

// Stmt is a code statement.
type Stmt struct {
	Assignment *Assignment `@@`
}

// Assignment is an assignment statement.
type Assignment struct {
	Left   string `@Ident`
	Equals string `@"="`
	Expr   *Expr  `@@`
	Semi   string `@";"`
}

// Expr is an expression.
type Expr struct {
	FuncCall *FuncCall `@@`
}

// FuncCall is a function call exporession.
type FuncCall struct {
	FuncName string   `@Ident`
	LParen   string   `@"("`
	Args     []string `@Ident ("," @Ident)*`
	RParen   string   `@")"`
}
