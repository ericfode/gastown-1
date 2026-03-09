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
	inPrompt bool // inside a prompt section (after system>, user>, etc.)
	promptBeforeFence bool // was in prompt mode before entering code fence
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
	"distill":  TokenDistill,
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

		if l.inPrompt {
			if err := l.lexPromptContent(); err != nil {
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
			} else if ch > 127 {
				// Unicode character outside of prompt/code fence context.
				// Consume as prompt text (common in comments that slipped through,
				// prompt fragments, etc.)
				start := l.pos
				for l.pos < len(l.input) && l.input[l.pos] != '\n' {
					l.advance()
				}
				l.emit(TokenPromptText, string(l.input[start:l.pos]))
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

	// Preserve promptBeforeFence if already set (e.g., prompt mode set it before returning)
	if !l.promptBeforeFence {
		l.promptBeforeFence = l.inPrompt
	}
	l.inPrompt = false
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
			l.inPrompt = l.promptBeforeFence
			l.promptBeforeFence = false
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

// lexPromptContent consumes prompt lines after a section tag (system>, user>, etc.).
// It collects lines as TokenPromptText until it sees a structural token at the start
// of a line: another section tag, cell close #/, code fence ```, etc.
func (l *Lexer) lexPromptContent() error {
	var lines []string

	for l.pos < len(l.input) {
		// Skip the newline at the start if we're right after the section tag
		if l.input[l.pos] == '\n' {
			l.advance()
			l.line++
			l.col = 1
			continue
		}

		// Check if this line starts with a structural token (exit prompt mode)
		lineStart := l.pos
		savedLine := l.line
		savedCol := l.col

		// Skip leading whitespace to peek at line content
		indent := 0
		for l.pos < len(l.input) && (l.input[l.pos] == ' ' || l.input[l.pos] == '\t') {
			indent++
			l.advance()
		}

		if l.pos >= len(l.input) {
			break
		}

		ch := l.input[l.pos]

		// Check for structural tokens that end prompt mode
		isStructural := false

		// ##/ or ## (molecule delimiters)
		if ch == '#' && l.peek(1) == '#' {
			isStructural = true
		}
		// #/ (cell end)
		if ch == '#' && l.peek(1) == '/' {
			isStructural = true
		}
		// # name : type (cell start) — # followed by space then ident
		if ch == '#' && l.peek(1) == ' ' {
			isStructural = true
		}
		// ``` (code fence) — emit accumulated prompt text, handle fence, continue prompt
		if ch == '`' && l.peek(1) == '`' && l.peek(2) == '`' {
			// Restore position, emit what we have, handle code fence, then continue
			l.pos = lineStart
			l.line = savedLine
			l.col = savedCol
			if len(lines) > 0 {
				l.emit(TokenPromptText, strings.Join(lines, "\n"))
				lines = nil
			}
			// Let the main loop handle the code fence; save prompt state
			l.promptBeforeFence = true
			l.inPrompt = false
			return nil
		}
		// Section tag: word followed by >
		if isIdentStart(ch) {
			peekPos := l.pos
			for peekPos < len(l.input) && isIdentChar(l.input[peekPos]) {
				peekPos++
			}
			if peekPos < len(l.input) && l.input[peekPos] == '>' {
				word := string(l.input[l.pos:peekPos])
				if _, ok := sectionTags[word]; ok {
					isStructural = true
				}
			}
		}
		// - ref (dependency declaration) at cell indent level
		if ch == '-' && l.peek(1) == ' ' && indent <= 4 {
			isStructural = true
		}
		// Keyword-based structural detection — require disambiguating context
		if !isStructural && isIdentStart(ch) {
			peekPos := l.pos
			for peekPos < len(l.input) && isIdentChar(l.input[peekPos]) {
				peekPos++
			}
			if peekPos < len(l.input) {
				word := string(l.input[l.pos:peekPos])
				rest := l.pos // position after the word
				_ = rest
				// Skip whitespace after word for lookahead
				afterWord := peekPos
				for afterWord < len(l.input) && (l.input[afterWord] == ' ' || l.input[afterWord] == '\t') {
					afterWord++
				}
				switch word {
				case "input":
					// "input param." is structural; "input validation" is not
					if afterWord+6 < len(l.input) && string(l.input[afterWord:afterWord+6]) == "param." {
						isStructural = true
					}
				case "map", "reduce":
					// "map #" is structural; "map of the city" is not
					if afterWord < len(l.input) && l.input[afterWord] == '#' {
						isStructural = true
					}
				case "meta":
					// "meta #" or "meta #/" is structural
					if afterWord < len(l.input) && l.input[afterWord] == '#' {
						isStructural = true
					}
				case "prompt":
					// "prompt@" is structural
					if afterWord < len(l.input) && l.input[afterWord] == '@' {
						isStructural = true
					}
				}
			}
		}

		if isStructural {
			// Restore position to line start — let normal lexer handle it
			l.pos = lineStart
			l.line = savedLine
			l.col = savedCol
			break
		}

		// Not structural — collect this line as prompt text
		l.pos = lineStart
		l.line = savedLine
		l.col = savedCol

		start := l.pos
		for l.pos < len(l.input) && l.input[l.pos] != '\n' {
			l.advance()
		}
		line := string(l.input[start:l.pos])
		lines = append(lines, line)

		if l.pos < len(l.input) {
			l.advance() // consume newline
			l.line++
			l.col = 1
		}
	}

	if len(lines) > 0 {
		l.emit(TokenPromptText, strings.Join(lines, "\n"))
	}

	l.inPrompt = false
	return nil
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
			// Enter prompt mode for tags with free-text content
			// (NOT each>, format>, vars>, squash> which have structured syntax)
			if word == "system" || word == "context" || word == "user" ||
				word == "think" || word == "examples" || word == "accept" ||
				word == "distill" {
				l.inPrompt = true
			}
			return
		}
	}

	// Check for prompt@
	if word == "prompt" && l.pos < len(l.input) && l.input[l.pos] == '@' {
		l.advance() // consume @
		l.emit(TokenPromptAt, "prompt@")
		// After the name token, the following lines are prompt content
		// We'll enter prompt mode after the next ident (the fragment name)
		// Actually, lex the name here then enter prompt mode
		for l.pos < len(l.input) && (l.input[l.pos] == ' ' || l.input[l.pos] == '\t') {
			l.advance()
		}
		if l.pos < len(l.input) && isIdentStart(l.input[l.pos]) {
			nameStart := l.pos
			for l.pos < len(l.input) && isIdentChar(l.input[l.pos]) {
				l.advance()
			}
			l.emit(TokenIdent, string(l.input[nameStart:l.pos]))
		}
		l.inPrompt = true
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
