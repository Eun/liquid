package render

import (
	"fmt"

	"github.com/osteele/liquid/expressions"
)

// Parse parses a source template. It returns an AST root, that can be evaluated.
func (s Config) Parse(source string) (ASTNode, error) {
	tokens := Scan(source, "")
	return s.parseChunks(tokens)
}

// Parse creates an AST from a sequence of Chunks.
func (s Config) parseChunks(chunks []Chunk) (ASTNode, error) { // nolint: gocyclo
	// a stack of control tag state, for matching nested {%if}{%endif%} etc.
	type frame struct {
		cd *blockDef  // saved local ccd
		cn *ASTBlock  // saved local cn
		ap *[]ASTNode // saved local ap
	}
	var (
		root      = &ASTSeq{}      // root of AST; will be returned
		ap        = &root.Children // newly-constructed nodes are appended here
		ccd       *blockDef        // current block definition
		ccn       *ASTBlock        // current block node
		stack     []frame          // stack of blocks
		rawTag    *ASTRaw          // current raw tag
		inComment = false
		inRaw     = false
	)
	for _, c := range chunks {
		switch {
		// The parser needs to know about comment and raw, because tags inside
		// needn't match each other e.g. {%comment%}{%if%}{%endcomment%}
		// TODO is this true?
		case inComment:
			if c.Type == TagChunkType && c.Name == "endcomment" {
				inComment = false
			}
		case inRaw:
			if c.Type == TagChunkType && c.Name == "endraw" {
				inRaw = false
			} else {
				rawTag.slices = append(rawTag.slices, c.Source)
			}
		case c.Type == ObjChunkType:
			expr, err := expressions.Parse(c.Args)
			if err != nil {
				return nil, err
			}
			*ap = append(*ap, &ASTObject{c, expr})
		case c.Type == TextChunkType:
			*ap = append(*ap, &ASTText{Chunk: c})
		case c.Type == TagChunkType:
			if cd, ok := s.findBlockDef(c.Name); ok {
				switch {
				case c.Name == "comment":
					inComment = true
				case c.Name == "raw":
					inRaw = true
					rawTag = &ASTRaw{}
					*ap = append(*ap, rawTag)
				case cd.requiresParent() && !cd.compatibleParent(ccd):
					suffix := ""
					if ccd != nil {
						suffix = "; immediate parent is " + ccd.name
					}
					return nil, fmt.Errorf("%s not inside %s%s", cd.name, cd.parent.name, suffix)
				case cd.isStartTag():
					stack = append(stack, frame{cd: ccd, cn: ccn, ap: ap})
					ccd, ccn = cd, &ASTBlock{Chunk: c, cd: cd}
					*ap = append(*ap, ccn)
					ap = &ccn.Body
				case cd.isBranchTag:
					n := &ASTBlock{Chunk: c, cd: cd}
					ccn.Branches = append(ccn.Branches, n)
					ap = &n.Body
				case cd.isEndTag:
					f := stack[len(stack)-1]
					stack = stack[:len(stack)-1]
					ccd, ccn, ap = f.cd, f.cn, f.ap
				}
			} else if td, ok := s.FindTagDefinition(c.Name); ok {
				f, err := td(c.Args)
				if err != nil {
					return nil, err
				}
				*ap = append(*ap, &ASTFunctional{c, f})
			} else {
				return nil, fmt.Errorf("unknown tag: %s", c.Name)
			}
		}
	}
	if ccd != nil {
		return nil, fmt.Errorf("unterminated %s tag at %s", ccd.name, ccn.SourceInfo)
	}
	if err := s.evaluateBuilders(root); err != nil {
		return nil, err
	}
	if len(root.Children) == 1 {
		return root.Children[0], nil
	}
	return root, nil
}

// nolint: gocyclo
func (s Config) evaluateBuilders(n ASTNode) error {
	switch n := n.(type) {
	case *ASTBlock:
		for _, child := range n.Body {
			if err := s.evaluateBuilders(child); err != nil {
				return err
			}
		}
		for _, branch := range n.Branches {
			if err := s.evaluateBuilders(branch); err != nil {
				return err
			}
		}
		cd, ok := s.findBlockDef(n.Name)
		if ok && cd.parser != nil {
			renderer, err := cd.parser(*n)
			if err != nil {
				return err
			}
			n.renderer = renderer
		}
	case *ASTSeq:
		for _, child := range n.Children {
			if error := s.evaluateBuilders(child); error != nil {
				return error
			}
		}
	}
	return nil
}
