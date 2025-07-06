// Copyright (c) 2025 Adam Snyder <https://armsnyder.com> and contributors
// SPDX-License-Identifier: MIT

package ast

import (
	"io"

	"github.com/alecthomas/participle/v2"
)

var parser *participle.Parser[File]

func init() {
	parser = participle.MustBuild[File]()
}

// Parse parses a .gdshader file into a tree of AST nodes.
func Parse(filename string, reader io.Reader) (*File, error) {
	return parser.Parse(filename, reader, participle.AllowTrailing(true))
}
