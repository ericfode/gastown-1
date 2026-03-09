package cell

import (
	"fmt"
	"os"
)

// ParseFile reads and parses a .cell file.
func ParseFile(path string) (*File, error) {
	data, err := os.ReadFile(path) //nolint:gosec // G304: path is user-provided CLI input
	if err != nil {
		return nil, fmt.Errorf("reading cell file: %w", err)
	}
	return Parse(string(data), path)
}

// Parse parses Cell language source code.
func Parse(input, filename string) (*File, error) {
	lexer := NewLexer(input, filename)
	tokens, err := lexer.Tokenize()
	if err != nil {
		return nil, err
	}
	p := &parser{tokens: tokens, filename: filename}
	return p.parseFile()
}

type parser struct {
	tokens   []Token
	pos      int
	filename string
}

func (p *parser) parseFile() (*File, error) {
	f := &File{}
	for !p.atEnd() {
		tok := p.current()
		if tok.Type != TokenIdent {
			return nil, p.errorf("expected 'cell' or 'recipe', got %s", tok)
		}
		switch tok.Value {
		case "cell":
			decl, err := p.parseCellDecl()
			if err != nil {
				return nil, err
			}
			f.Cells = append(f.Cells, decl)
		case "recipe":
			decl, err := p.parseRecipeDecl()
			if err != nil {
				return nil, err
			}
			f.Recipes = append(f.Recipes, decl)
		default:
			return nil, p.errorf("expected 'cell' or 'recipe', got %q", tok.Value)
		}
	}
	return f, nil
}

func (p *parser) parseCellDecl() (*CellDecl, error) {
	pos := p.current().Pos
	p.advance() // skip "cell"

	name, err := p.expectIdent()
	if err != nil {
		return nil, err
	}

	if err := p.expect(TokenLBrace); err != nil {
		return nil, err
	}

	decl := &CellDecl{Name: name, Pos: pos}

	for !p.check(TokenRBrace) && !p.atEnd() {
		key, err := p.expectIdent()
		if err != nil {
			return nil, err
		}
		if err := p.expect(TokenColon); err != nil {
			return nil, err
		}

		switch key {
		case "type":
			decl.Type, err = p.expectIdentOrString()
			if err != nil {
				return nil, err
			}
		case "prompt":
			decl.Prompt, err = p.expectString()
			if err != nil {
				return nil, err
			}
		case "refs":
			decl.Refs, err = p.parseIdentList()
			if err != nil {
				return nil, err
			}
		case "oracle":
			decl.Oracle, err = p.expectIdentOrString()
			if err != nil {
				return nil, err
			}
		default:
			return nil, p.errorf("unknown cell field %q (expected type, prompt, refs, or oracle)", key)
		}
	}

	if err := p.expect(TokenRBrace); err != nil {
		return nil, err
	}

	return decl, nil
}

func (p *parser) parseRecipeDecl() (*RecipeDecl, error) {
	pos := p.current().Pos
	p.advance() // skip "recipe"

	name, err := p.expectIdent()
	if err != nil {
		return nil, err
	}

	// Parse parameter list
	if err := p.expect(TokenLParen); err != nil {
		return nil, err
	}
	var params []string
	for !p.check(TokenRParen) && !p.atEnd() {
		param, err := p.expectIdent()
		if err != nil {
			return nil, err
		}
		params = append(params, param)
		if !p.check(TokenRParen) {
			if err := p.expect(TokenComma); err != nil {
				return nil, err
			}
		}
	}
	if err := p.expect(TokenRParen); err != nil {
		return nil, err
	}

	// Parse body
	if err := p.expect(TokenLBrace); err != nil {
		return nil, err
	}

	decl := &RecipeDecl{Name: name, Params: params, Pos: pos}

	for !p.check(TokenRBrace) && !p.atEnd() {
		stmt, err := p.parseStatement()
		if err != nil {
			return nil, err
		}
		decl.Body = append(decl.Body, stmt)
	}

	if err := p.expect(TokenRBrace); err != nil {
		return nil, err
	}

	return decl, nil
}

func (p *parser) parseStatement() (*Statement, error) {
	pos := p.current().Pos
	name, err := p.expectIdent()
	if err != nil {
		return nil, err
	}

	// Check if this is an assignment (name = call) or a bare call (name(...))
	if p.check(TokenEquals) {
		p.advance() // skip =
		call, err := p.parseCall()
		if err != nil {
			return nil, err
		}
		return &Statement{
			Assignment: &Assignment{Name: name, Call: call},
			Pos:        pos,
		}, nil
	}

	// Must be a bare call
	if !p.check(TokenLParen) {
		return nil, p.errorf("expected '=' or '(' after %q", name)
	}
	call, err := p.parseCallWithName(name, pos)
	if err != nil {
		return nil, err
	}
	return &Statement{Call: call, Pos: pos}, nil
}

func (p *parser) parseCall() (*Call, error) {
	pos := p.current().Pos
	name, err := p.expectIdent()
	if err != nil {
		return nil, err
	}
	return p.parseCallWithName(name, pos)
}

func (p *parser) parseCallWithName(name string, pos Position) (*Call, error) {
	if err := p.expect(TokenLParen); err != nil {
		return nil, err
	}

	var args []Arg
	for !p.check(TokenRParen) && !p.atEnd() {
		arg, err := p.parseArg()
		if err != nil {
			return nil, err
		}
		args = append(args, arg)
		if !p.check(TokenRParen) {
			if err := p.expect(TokenComma); err != nil {
				return nil, err
			}
		}
	}

	if err := p.expect(TokenRParen); err != nil {
		return nil, err
	}

	return &Call{Name: name, Args: args, Pos: pos}, nil
}

func (p *parser) parseArg() (Arg, error) {
	tok := p.current()
	switch tok.Type {
	case TokenString:
		p.advance()
		return Arg{Str: tok.Value}, nil
	case TokenIdent:
		p.advance()
		return Arg{Ident: tok.Value}, nil
	case TokenLBracket:
		return p.parseListArg()
	case TokenLBrace:
		return p.parseObjectArg()
	default:
		return Arg{}, p.errorf("expected argument, got %s", tok)
	}
}

func (p *parser) parseListArg() (Arg, error) {
	p.advance() // skip [
	var items []Arg
	for !p.check(TokenRBracket) && !p.atEnd() {
		item, err := p.parseArg()
		if err != nil {
			return Arg{}, err
		}
		items = append(items, item)
		if !p.check(TokenRBracket) {
			if err := p.expect(TokenComma); err != nil {
				return Arg{}, err
			}
		}
	}
	if err := p.expect(TokenRBracket); err != nil {
		return Arg{}, err
	}
	return Arg{List: items}, nil
}

func (p *parser) parseObjectArg() (Arg, error) {
	p.advance() // skip {
	var fields []Field
	for !p.check(TokenRBrace) && !p.atEnd() {
		key, err := p.expectIdent()
		if err != nil {
			return Arg{}, err
		}
		if err := p.expect(TokenColon); err != nil {
			return Arg{}, err
		}
		val, err := p.parseArg()
		if err != nil {
			return Arg{}, err
		}
		fields = append(fields, Field{Key: key, Value: val})
		if !p.check(TokenRBrace) {
			if err := p.expect(TokenComma); err != nil {
				return Arg{}, err
			}
		}
	}
	if err := p.expect(TokenRBrace); err != nil {
		return Arg{}, err
	}
	return Arg{Object: fields}, nil
}

func (p *parser) parseIdentList() ([]string, error) {
	if err := p.expect(TokenLBracket); err != nil {
		return nil, err
	}
	var items []string
	for !p.check(TokenRBracket) && !p.atEnd() {
		name, err := p.expectIdent()
		if err != nil {
			return nil, err
		}
		items = append(items, name)
		if !p.check(TokenRBracket) {
			if err := p.expect(TokenComma); err != nil {
				return nil, err
			}
		}
	}
	if err := p.expect(TokenRBracket); err != nil {
		return nil, err
	}
	return items, nil
}

// Helper methods

func (p *parser) current() Token {
	if p.pos >= len(p.tokens) {
		return Token{Type: TokenEOF, Pos: Position{File: p.filename}}
	}
	return p.tokens[p.pos]
}

func (p *parser) advance() {
	if p.pos < len(p.tokens) {
		p.pos++
	}
}

func (p *parser) atEnd() bool {
	return p.pos >= len(p.tokens) || p.tokens[p.pos].Type == TokenEOF
}

func (p *parser) check(t TokenType) bool {
	return p.current().Type == t
}

func (p *parser) expect(t TokenType) error {
	tok := p.current()
	if tok.Type != t {
		return p.errorf("expected %s, got %s", t, tok)
	}
	p.advance()
	return nil
}

func (p *parser) expectIdent() (string, error) {
	tok := p.current()
	if tok.Type != TokenIdent {
		return "", p.errorf("expected identifier, got %s", tok)
	}
	p.advance()
	return tok.Value, nil
}

func (p *parser) expectString() (string, error) {
	tok := p.current()
	if tok.Type != TokenString {
		return "", p.errorf("expected string, got %s", tok)
	}
	p.advance()
	return tok.Value, nil
}

func (p *parser) expectIdentOrString() (string, error) {
	tok := p.current()
	if tok.Type != TokenIdent && tok.Type != TokenString {
		return "", p.errorf("expected identifier or string, got %s", tok)
	}
	p.advance()
	return tok.Value, nil
}

func (p *parser) errorf(format string, args ...any) error {
	pos := p.current().Pos
	return fmt.Errorf("%s: %s", pos, fmt.Sprintf(format, args...))
}
