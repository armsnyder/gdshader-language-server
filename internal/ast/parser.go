package ast

import (
	"io"

	"github.com/alecthomas/participle/v2"
)

var parser *participle.Parser[File]

func init() {
	parser = participle.MustBuild[File]()
}

func Parse(filename string, reader io.Reader) (*File, error) {
	return parser.Parse(filename, reader, participle.AllowTrailing(true))
}
