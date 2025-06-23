package ast

import "github.com/alecthomas/participle/v2/lexer"

type File struct {
	Pos          lexer.Position
	Declarations []*Declaration `@@*`
}

type Declaration struct {
	UniformDecl  *UniformDecl  `@@`
	FunctionDecl *FunctionDecl `| @@`
}

type UniformDecl struct {
	Type string `"uniform" @Ident`
	Name string `@Ident ";"`
}

type FunctionDecl struct {
	ReturnType string     `@Ident`
	Name       string     `@Ident "(" ")"`
	Body       *BlockStmt `@@`
}

type BlockStmt struct {
	Stmts []*Stmt `"{" @@* "}"`
}

type Stmt struct {
	Assignment *Assignment `@@`
}

type Assignment struct {
	Left   string `@Ident`
	Equals string `@"="`
	Expr   *Expr  `@@`
	Semi   string `@";"`
}

type Expr struct {
	FuncCall *FuncCall `@@`
}

type FuncCall struct {
	FuncName string   `@Ident`
	LParen   string   `@"("`
	Args     []string `@Ident ("," @Ident)*`
	RParen   string   `@")"`
}
