package render

import (
	"io"

	"github.com/osteele/liquid/expression"
	"github.com/osteele/liquid/parser"
)

// Node is a node of the render tree.
type Node interface {
}

// BlockNode represents a {% tag %}…{% endtag %}.
type BlockNode struct {
	parser.Token
	renderer func(io.Writer, Context) error
	Body     []Node
	Branches []*BlockNode
}

// RawNode holds the text between the start and end of a raw tag.
type RawNode struct {
	slices []string
}

// TagNode renders itself via a render function that is created during parsing.
type TagNode struct {
	parser.Token
	renderer func(io.Writer, Context) error
}

// TextNode is a text chunk, that is rendered verbatim.
type TextNode struct {
	parser.Token
}

// ObjectNode is an {{ object }} object.
type ObjectNode struct {
	parser.Token
	expr expression.Expression
}

// SeqNode is a sequence of nodes.
type SeqNode struct {
	Children []Node
}
