package cell

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// TokenType identifies the kind of token.
type TokenType int

const (
	TokenEOF TokenType = iota
	TokenIdent         // identifiers and keywords
	TokenString        // "..." or """..."""
	TokenLBrace        // {
	TokenRBrace        // }
	TokenLBracket      // [
	TokenRBracket      // ]
	TokenLParen        // (
	TokenRParen        // )
	TokenColon         // :
	TokenComma         // ,
	TokenEquals        // =
	TokenError         // lexer error
)

var tokenNames = map[TokenType]string{
	TokenEOF:      "EOF",
	TokenIdent:    "identifier",
	TokenString:   "string",
	TokenLBrace:   "{",
	TokenRBrace:   "}",
	TokenLBracket: "[",
	TokenRBracket: "]",
	TokenLParen:   "(",
	TokenRParen:   ")",
	TokenColon:    ":",
	TokenComma:    ",",
	TokenEquals:   "=",
	TokenError:    "error",
}

func (t TokenType) String() string {
	if s, ok := tokenNames[t]; ok {
		return s
	}
	return fmt.Sprintf("TokenType(%d)", t)
}

// Token is a lexical token.
type Token struct {
	Type    TokenType
	Value   string
	Pos     Position
}

func (t Token) String() string {
	if t.Type == TokenString {
		return fmt.Sprintf("%s(%q)", t.Type, t.Value)
	}
	return fmt.Sprintf("%s(%s)", t.Type, t.Value)
}

// Lexer tokenizes Cell language source code.
type Lexer struct {
	input    string
	filename string
	pos      int // current byte position
	line     int
	col      int
	tokens   []Token
}

// NewLexer creates a new lexer for the given source.
func NewLexer(input, filename string) *Lexer {
	return &Lexer{
		input:    input,
		filename: filename,
		line:     1,
		col:      1,
	}
}

// Tokenize lexes the entire input and returns the token stream.
func (l *Lexer) Tokenize() ([]Token, error) {
	for {
		tok, err := l.next()
		if err != nil {
			return nil, err
		}
		l.tokens = append(l.tokens, tok)
		if tok.Type == TokenEOF {
			break
		}
	}
	return l.tokens, nil
}

func (l *Lexer) next() (Token, error) {
	l.skipWhitespaceAndComments()

	if l.pos >= len(l.input) {
		return Token{Type: TokenEOF, Pos: l.position()}, nil
	}

	pos := l.position()
	ch := l.peek()

	switch ch {
	case '{':
		l.advance()
		return Token{Type: TokenLBrace, Value: "{", Pos: pos}, nil
	case '}':
		l.advance()
		return Token{Type: TokenRBrace, Value: "}", Pos: pos}, nil
	case '[':
		l.advance()
		return Token{Type: TokenLBracket, Value: "[", Pos: pos}, nil
	case ']':
		l.advance()
		return Token{Type: TokenRBracket, Value: "]", Pos: pos}, nil
	case '(':
		l.advance()
		return Token{Type: TokenLParen, Value: "(", Pos: pos}, nil
	case ')':
		l.advance()
		return Token{Type: TokenRParen, Value: ")", Pos: pos}, nil
	case ':':
		l.advance()
		return Token{Type: TokenColon, Value: ":", Pos: pos}, nil
	case ',':
		l.advance()
		return Token{Type: TokenComma, Value: ",", Pos: pos}, nil
	case '=':
		l.advance()
		return Token{Type: TokenEquals, Value: "=", Pos: pos}, nil
	case '"':
		return l.lexString(pos)
	default:
		if isIdentStart(ch) {
			return l.lexIdent(pos), nil
		}
		return Token{}, fmt.Errorf("%s: unexpected character %q", pos.String(), string(ch))
	}
}

func (l *Lexer) lexIdent(pos Position) Token {
	start := l.pos
	for l.pos < len(l.input) {
		ch := l.peek()
		if isIdentCont(ch) {
			l.advance()
		} else {
			break
		}
	}
	return Token{Type: TokenIdent, Value: l.input[start:l.pos], Pos: pos}
}

func (l *Lexer) lexString(pos Position) (Token, error) {
	// Check for triple-quoted string """..."""
	if l.pos+2 < len(l.input) && l.input[l.pos:l.pos+3] == `"""` {
		return l.lexTripleString(pos)
	}
	return l.lexSimpleString(pos)
}

func (l *Lexer) lexSimpleString(pos Position) (Token, error) {
	l.advance() // skip opening "
	var sb strings.Builder
	for l.pos < len(l.input) {
		ch := l.peek()
		if ch == '\\' {
			l.advance()
			if l.pos >= len(l.input) {
				return Token{}, fmt.Errorf("%s: unterminated string escape", pos.String())
			}
			esc := l.peek()
			l.advance()
			switch esc {
			case 'n':
				sb.WriteByte('\n')
			case 't':
				sb.WriteByte('\t')
			case '\\':
				sb.WriteByte('\\')
			case '"':
				sb.WriteByte('"')
			default:
				sb.WriteByte('\\')
				sb.WriteRune(esc)
			}
		} else if ch == '"' {
			l.advance()
			return Token{Type: TokenString, Value: sb.String(), Pos: pos}, nil
		} else if ch == '\n' {
			return Token{}, fmt.Errorf("%s: unterminated string literal (use triple-quotes for multiline)", pos.String())
		} else {
			l.advance()
			sb.WriteRune(ch)
		}
	}
	return Token{}, fmt.Errorf("%s: unterminated string literal", pos.String())
}

func (l *Lexer) lexTripleString(pos Position) (Token, error) {
	// Skip opening """
	l.advance()
	l.advance()
	l.advance()

	var sb strings.Builder
	for l.pos < len(l.input) {
		if l.pos+2 < len(l.input) && l.input[l.pos:l.pos+3] == `"""` {
			l.advance()
			l.advance()
			l.advance()
			// Trim leading/trailing newlines and dedent
			content := dedentTripleString(sb.String())
			return Token{Type: TokenString, Value: content, Pos: pos}, nil
		}
		ch := l.peek()
		l.advance()
		sb.WriteRune(ch)
	}
	return Token{}, fmt.Errorf("%s: unterminated triple-quoted string", pos.String())
}

func (l *Lexer) skipWhitespaceAndComments() {
	for l.pos < len(l.input) {
		ch := l.peek()
		if ch == '#' {
			// Skip line comment
			for l.pos < len(l.input) && l.peek() != '\n' {
				l.advance()
			}
		} else if unicode.IsSpace(ch) {
			l.advance()
		} else {
			break
		}
	}
}

func (l *Lexer) peek() rune {
	r, _ := utf8.DecodeRuneInString(l.input[l.pos:])
	return r
}

func (l *Lexer) advance() {
	r, size := utf8.DecodeRuneInString(l.input[l.pos:])
	if r == '\n' {
		l.line++
		l.col = 1
	} else {
		l.col++
	}
	l.pos += size
}

func (l *Lexer) position() Position {
	return Position{File: l.filename, Line: l.line, Column: l.col}
}

func (p Position) String() string {
	if p.File != "" {
		return fmt.Sprintf("%s:%d:%d", p.File, p.Line, p.Column)
	}
	return fmt.Sprintf("%d:%d", p.Line, p.Column)
}

func isIdentStart(r rune) bool {
	return unicode.IsLetter(r) || r == '_'
}

func isIdentCont(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-'
}

// dedentTripleString removes common leading whitespace from a triple-quoted string.
func dedentTripleString(s string) string {
	// Trim leading newline
	if len(s) > 0 && s[0] == '\n' {
		s = s[1:]
	}
	// Trim trailing whitespace + newline
	s = strings.TrimRight(s, " \t\n\r")

	lines := strings.Split(s, "\n")
	if len(lines) == 0 {
		return ""
	}

	// Find minimum indent (ignoring empty lines)
	minIndent := -1
	for _, line := range lines {
		trimmed := strings.TrimLeft(line, " \t")
		if trimmed == "" {
			continue
		}
		indent := len(line) - len(trimmed)
		if minIndent < 0 || indent < minIndent {
			minIndent = indent
		}
	}
	if minIndent <= 0 {
		return strings.Join(lines, "\n")
	}

	// Remove common indent
	for i, line := range lines {
		if len(line) >= minIndent {
			lines[i] = line[minIndent:]
		}
	}
	return strings.Join(lines, "\n")
}
