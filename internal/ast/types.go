// Copyright (c) 2025 Adam Snyder <https://armsnyder.com> and contributors
// SPDX-License-Identifier: MIT

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
