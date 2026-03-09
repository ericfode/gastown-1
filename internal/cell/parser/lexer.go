package parser

import (
	"fmt"
	"strings"
	"unicode"
)

// Lexer tokenizes Cell source code.
type Lexer struct {
	input   []rune
	pos     int
	line    int
	col     int
	tokens  []Token
	inCodeFence bool // inside ``` ... ```
	codeFenceType string // e.g., "oracle", "sh"
}

// keywords maps keyword strings to token types.
var keywords = map[string]TokenType{
	"meta":           TokenMeta,
	"map":            TokenMap,
	"reduce":         TokenReduce,
	"over":           TokenOver,
	"as":             TokenAs,
	"with":           TokenWith,
	"input":          TokenInput,
	"preset":         TokenPreset,
	"recipe":         TokenRecipe,
	"import":         TokenImport,
	"apply":          TokenApply,
	"where":          TokenWhere,
	"and":            TokenAnd,
	"or":             TokenOr,
	"not":            TokenNot,
	"in":             TokenIn,
	"true":           TokenTrue,
	"false":          TokenFalse,
	"null":           TokenNull,
	"if":             TokenIf,
	"for":            TokenFor,
	"required":       TokenRequired,
	"required_unless": TokenRequiredUnless,
	"default":        TokenDefault,
	"contains":       TokenContains,
	"matches":        TokenMatches,
	"typeof":         TokenTypeof,
	"len":            TokenLen,
	"json_parse":     TokenJsonParse,
	"keys_present":   TokenKeysPresent,
	"assert":         TokenAssert,
	"score":          TokenScore,
	"reject":         TokenReject,
	"accept":         TokenAccept,
	"str":            TokenStr,
	"number":         TokenTypeNumber,
	"boolean":        TokenBoolean,
	"json":           TokenJson,
	"enum":           TokenEnum,
	"mol":            TokenMol,
}

// sectionTags maps section tag names to token types.
var sectionTags = map[string]TokenType{
	"system":   TokenSystem,
	"context":  TokenContext,
	"user":     TokenUser,
	"think":    TokenThink,
	"examples": TokenExamples,
	"format":   TokenFormat,
	"accept":   TokenAccept,
	"each":     TokenEach,
	"vars":     TokenVars,
	"squash":   TokenSquash,
}

// Lex tokenizes the input source code.
func Lex(source string) ([]Token, error) {
	l := &Lexer{
		input: []rune(source),
		line:  1,
		col:   1,
	}
	if err := l.lex(); err != nil {
		return nil, err
	}
	return l.tokens, nil
}

func (l *Lexer) lex() error {
	for l.pos < len(l.input) {
		if l.inCodeFence {
			if err := l.lexCodeFenceContent(); err != nil {
				return err
			}
			continue
		}

		ch := l.input[l.pos]

		// Skip whitespace (but not newlines)
		if ch == ' ' || ch == '\t' || ch == '\r' {
			l.advance()
			continue
		}

		if ch == '\n' {
			l.emit(TokenNewline, "\n")
			l.advance()
			l.line++
			l.col = 1
			continue
		}

		// Comments: -- ...
		if ch == '-' && l.peek(1) == '-' {
			l.lexComment()
			continue
		}

		// Code fence: ```
		if ch == '`' && l.peek(1) == '`' && l.peek(2) == '`' {
			l.lexCodeFenceStart()
			continue
		}

		// Double braces: {{ and }}
		if ch == '{' && l.peek(1) == '{' {
			l.emit(TokenDoubleLBrace, "{{")
			l.advance()
			l.advance()
			continue
		}
		if ch == '}' && l.peek(1) == '}' {
			l.emit(TokenDoubleRBrace, "}}")
			l.advance()
			l.advance()
			continue
		}

		// ### or more: markdown heading in prompt text, not structural
		if ch == '#' && l.peek(1) == '#' && l.peek(2) == '#' {
			// Count all consecutive hashes
			start := l.pos
			for l.pos < len(l.input) && l.input[l.pos] == '#' {
				l.advance()
			}
			l.emit(TokenIdent, string(l.input[start:l.pos]))
			continue
		}

		// ## and ##/ (molecule delimiters)
		if ch == '#' && l.peek(1) == '#' {
			if l.peek(2) == '/' {
				l.emit(TokenDoubleHashSlash, "##/")
				l.advance()
				l.advance()
				l.advance()
			} else {
				l.emit(TokenDoubleHash, "##")
				l.advance()
				l.advance()
			}
			continue
		}

		// #/ (cell end)
		if ch == '#' && l.peek(1) == '/' {
			l.emit(TokenHashSlash, "#/")
			l.advance()
			l.advance()
			continue
		}

		// # (cell start)
		if ch == '#' {
			l.emit(TokenHash, "#")
			l.advance()
			continue
		}

		// -> arrow
		if ch == '-' && l.peek(1) == '>' {
			l.emit(TokenArrow, "->")
			l.advance()
			l.advance()
			continue
		}

		// => fat arrow
		if ch == '=' && l.peek(1) == '>' {
			l.emit(TokenFatArrow, "=>")
			l.advance()
			l.advance()
			continue
		}

		// == != <= >=
		if ch == '=' && l.peek(1) == '=' {
			l.emit(TokenEqEq, "==")
			l.advance()
			l.advance()
			continue
		}
		if ch == '!' && l.peek(1) == '=' {
			l.emit(TokenNotEq, "!=")
			l.advance()
			l.advance()
			continue
		}
		if ch == '<' && l.peek(1) == '=' {
			l.emit(TokenLTEq, "<=")
			l.advance()
			l.advance()
			continue
		}
		if ch == '>' && l.peek(1) == '=' {
			l.emit(TokenGTEq, ">=")
			l.advance()
			l.advance()
			continue
		}

		// Single character tokens
		switch ch {
		case '{':
			l.emit(TokenLBrace, "{")
			l.advance()
		case '}':
			l.emit(TokenRBrace, "}")
			l.advance()
		case '[':
			l.emit(TokenLBracket, "[")
			l.advance()
		case ']':
			l.emit(TokenRBracket, "]")
			l.advance()
		case '(':
			l.emit(TokenLParen, "(")
			l.advance()
		case ')':
			l.emit(TokenRParen, ")")
			l.advance()
		case ',':
			l.emit(TokenComma, ",")
			l.advance()
		case ':':
			l.emit(TokenColon, ":")
			l.advance()
		case '.':
			l.emit(TokenDot, ".")
			l.advance()
		case '|':
			l.emit(TokenPipe, "|")
			l.advance()
		case '@':
			l.emit(TokenAt, "@")
			l.advance()
		case '-':
			l.emit(TokenDash, "-")
			l.advance()
		case '?':
			l.emit(TokenQuestion, "?")
			l.advance()
		case '=':
			l.emit(TokenEquals, "=")
			l.advance()
		case '!':
			// Check for graph operations: !add, !drop, etc.
			if l.lexBangOp() {
				continue
			}
			l.emit(TokenBang, "!")
			l.advance()
		case ';':
			l.emit(TokenSemicolon, ";")
			l.advance()
		case '<':
			l.emit(TokenLT, "<")
			l.advance()
		case '>':
			l.emit(TokenGT, ">")
			l.advance()
		case '"':
			if err := l.lexString(); err != nil {
				return err
			}
		case '*':
			l.emit(TokenIdent, "*")
			l.advance()
		case '/':
			// Standalone / (not part of #/ or ##/) — treat as ident char in prompt text
			l.emit(TokenIdent, "/")
			l.advance()
		case '+':
			// +0.3 style scores — lex as number
			if l.peek(1) >= '0' && l.peek(1) <= '9' {
				l.lexNumber()
			} else {
				l.emit(TokenIdent, "+")
				l.advance()
			}
		default:
			if unicode.IsDigit(ch) {
				l.lexNumber()
			} else if isIdentStart(ch) {
				l.lexIdentOrKeyword()
			} else {
				return l.errorf("unexpected character: %c", ch)
			}
		}
	}

	l.emit(TokenEOF, "")
	return nil
}

func (l *Lexer) lexComment() {
	start := l.pos
	for l.pos < len(l.input) && l.input[l.pos] != '\n' {
		l.advance()
	}
	l.emit(TokenComment, string(l.input[start:l.pos]))
}

func (l *Lexer) lexCodeFenceStart() {
	l.emit(TokenCodeFence, "```")
	l.advance() // `
	l.advance() // `
	l.advance() // `

	// Skip whitespace after ```
	for l.pos < len(l.input) && (l.input[l.pos] == ' ' || l.input[l.pos] == '\t') {
		l.advance()
	}

	// Read optional tag (e.g., "oracle", "sh")
	if l.pos < len(l.input) && l.input[l.pos] != '\n' {
		start := l.pos
		for l.pos < len(l.input) && l.input[l.pos] != '\n' && l.input[l.pos] != ' ' {
			l.advance()
		}
		tag := string(l.input[start:l.pos])
		l.codeFenceType = tag
		l.emit(TokenCodeFenceTag, tag)
	}

	// Skip to end of line
	for l.pos < len(l.input) && l.input[l.pos] != '\n' {
		l.advance()
	}
	if l.pos < len(l.input) {
		l.advance() // consume newline
		l.line++
		l.col = 1
	}

	l.inCodeFence = true
}

func (l *Lexer) lexCodeFenceContent() error {
	// Collect lines until closing ```
	var lines []string
	for l.pos < len(l.input) {
		// Check for closing ```
		lineStart := l.pos
		// Skip leading whitespace
		for l.pos < len(l.input) && (l.input[l.pos] == ' ' || l.input[l.pos] == '\t') {
			l.advance()
		}
		if l.pos+2 < len(l.input) && l.input[l.pos] == '`' && l.input[l.pos+1] == '`' && l.input[l.pos+2] == '`' {
			// Found closing fence
			if len(lines) > 0 {
				l.emit(TokenPromptText, strings.Join(lines, "\n"))
			}
			l.emit(TokenCodeFence, "```")
			l.advance()
			l.advance()
			l.advance()
			// Skip rest of line
			for l.pos < len(l.input) && l.input[l.pos] != '\n' {
				l.advance()
			}
			if l.pos < len(l.input) {
				l.advance()
				l.line++
				l.col = 1
			}
			l.inCodeFence = false
			l.codeFenceType = ""
			return nil
		}

		// Not a closing fence — collect the line
		l.pos = lineStart
		start := l.pos
		for l.pos < len(l.input) && l.input[l.pos] != '\n' {
			l.advance()
		}
		lines = append(lines, string(l.input[start:l.pos]))
		if l.pos < len(l.input) {
			l.advance()
			l.line++
			l.col = 1
		}
	}
	return l.errorf("unterminated code fence")
}

func (l *Lexer) lexString() error {
	l.advance() // consume opening "
	var sb strings.Builder
	for l.pos < len(l.input) {
		ch := l.input[l.pos]
		if ch == '\\' {
			l.advance()
			if l.pos >= len(l.input) {
				return l.errorf("unterminated string escape")
			}
			esc := l.input[l.pos]
			switch esc {
			case '"':
				sb.WriteRune('"')
			case '\\':
				sb.WriteRune('\\')
			case 'n':
				sb.WriteRune('\n')
			case 't':
				sb.WriteRune('\t')
			case 'r':
				sb.WriteRune('\r')
			default:
				sb.WriteRune('\\')
				sb.WriteRune(esc)
			}
			l.advance()
			continue
		}
		if ch == '"' {
			l.advance() // consume closing "
			l.emit(TokenString, sb.String())
			return nil
		}
		if ch == '\n' {
			return l.errorf("unterminated string (newline in string)")
		}
		sb.WriteRune(ch)
		l.advance()
	}
	return l.errorf("unterminated string")
}

func (l *Lexer) lexNumber() {
	start := l.pos
	// Handle leading +
	if l.pos < len(l.input) && l.input[l.pos] == '+' {
		l.advance()
	}
	for l.pos < len(l.input) && unicode.IsDigit(l.input[l.pos]) {
		l.advance()
	}
	if l.pos < len(l.input) && l.input[l.pos] == '.' {
		l.advance()
		for l.pos < len(l.input) && unicode.IsDigit(l.input[l.pos]) {
			l.advance()
		}
	}
	l.emit(TokenNumber, string(l.input[start:l.pos]))
}

func (l *Lexer) lexIdentOrKeyword() {
	start := l.pos
	for l.pos < len(l.input) && isIdentChar(l.input[l.pos]) {
		l.advance()
	}
	word := string(l.input[start:l.pos])

	// Check for section tags: word followed by >
	if l.pos < len(l.input) && l.input[l.pos] == '>' {
		if tt, ok := sectionTags[word]; ok {
			l.advance() // consume >
			l.emit(tt, word+">")
			return
		}
	}

	// Check for prompt@
	if word == "prompt" && l.pos < len(l.input) && l.input[l.pos] == '@' {
		l.advance() // consume @
		l.emit(TokenPromptAt, "prompt@")
		return
	}

	// Check for keywords
	if tt, ok := keywords[word]; ok {
		l.emit(tt, word)
		return
	}

	l.emit(TokenIdent, word)
}

func (l *Lexer) lexBangOp() bool {
	// Check if ! is followed by a known operation keyword
	ops := map[string]TokenType{
		"add":    TokenOpAdd,
		"drop":   TokenOpDrop,
		"wire":   TokenOpWire,
		"cut":    TokenOpCut,
		"split":  TokenOpSplit,
		"merge":  TokenOpMerge,
		"refine": TokenOpRefine,
		"seed":   TokenOpSeed,
	}

	// Look ahead for the keyword
	savedPos := l.pos
	savedCol := l.col
	l.advance() // skip !

	start := l.pos
	for l.pos < len(l.input) && isIdentChar(l.input[l.pos]) {
		l.advance()
	}
	word := string(l.input[start:l.pos])

	if tt, ok := ops[word]; ok {
		l.emit(tt, "!"+word)
		return true
	}

	// Not a bang op — restore
	l.pos = savedPos
	l.col = savedCol
	return false
}

func (l *Lexer) emit(tt TokenType, value string) {
	l.tokens = append(l.tokens, Token{
		Type:  tt,
		Value: value,
		Line:  l.line,
		Col:   l.col - len([]rune(value)),
	})
}

func (l *Lexer) advance() {
	if l.pos < len(l.input) {
		l.pos++
		l.col++
	}
}

func (l *Lexer) peek(offset int) rune {
	idx := l.pos + offset
	if idx >= len(l.input) {
		return 0
	}
	return l.input[idx]
}

func (l *Lexer) errorf(format string, args ...any) error {
	return fmt.Errorf("lexer error at %d:%d: %s", l.line, l.col, fmt.Sprintf(format, args...))
}

func isIdentStart(ch rune) bool {
	return unicode.IsLetter(ch) || ch == '_'
}

func isIdentChar(ch rune) bool {
	return unicode.IsLetter(ch) || unicode.IsDigit(ch) || ch == '_' || ch == '-'
}
