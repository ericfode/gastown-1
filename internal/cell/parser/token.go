// Package parser implements a recursive descent parser for the Cell language.
//
// Cell is a context-free DSL for describing reactive computation graphs
// where cells are agents, wires are typed data flows, and oracles gate quality.
package parser

import "fmt"

// TokenType identifies the kind of token.
type TokenType int

const (
	// Structural tokens
	TokenEOF TokenType = iota
	TokenNewline
	TokenIndent // significant indentation in prompt lines

	// Delimiters
	TokenHash          // #
	TokenHashSlash     // #/
	TokenDoubleHash    // ##
	TokenDoubleHashSlash // ##/
	TokenLBrace        // {
	TokenRBrace        // }
	TokenLBracket      // [
	TokenRBracket      // ]
	TokenLParen        // (
	TokenRParen        // )
	TokenComma         // ,
	TokenColon         // :
	TokenDot           // .
	TokenPipe          // |
	TokenArrow         // ->
	TokenFatArrow      // =>
	TokenAt            // @
	TokenDash          // -
	TokenQuestion      // ?
	TokenEquals        // =
	TokenBang          // !
	TokenSemicolon     // ;
	TokenDoubleLBrace  // {{
	TokenDoubleRBrace  // }}

	// Comparison operators
	TokenEqEq   // ==
	TokenNotEq  // !=
	TokenLT     // <
	TokenGT     // >
	TokenLTEq   // <=
	TokenGTEq   // >=

	// Literals
	TokenIdent   // identifiers
	TokenString  // "..."
	TokenNumber  // 42, 3.14
	TokenComment // -- ...

	// Keywords
	TokenMolecule   // ## (molecule start, contextual)
	TokenCell       // # (cell start, contextual)
	TokenMeta       // meta
	TokenMap        // map
	TokenReduce     // reduce
	TokenOver       // over
	TokenAs         // as
	TokenWith       // with
	TokenInput      // input
	TokenPreset     // preset
	TokenRecipe     // recipe
	TokenImport     // import
	TokenApply      // apply
	TokenWhere      // where
	TokenAnd        // and
	TokenOr         // or
	TokenNot        // not
	TokenIn         // in
	TokenTrue       // true
	TokenFalse      // false
	TokenNull       // null
	TokenIf         // if
	TokenFor        // for
	TokenRequired   // required
	TokenRequiredUnless // required_unless
	TokenDefault    // default
	TokenContains   // contains
	TokenMatches    // matches
	TokenTypeof     // typeof
	TokenLen        // len
	TokenPromptAt   // prompt@

	// Section tags
	TokenSystem   // system>
	TokenContext  // context>
	TokenUser     // user>
	TokenThink    // think>
	TokenExamples // examples>
	TokenFormat   // format>
	TokenAccept   // accept>
	TokenEach     // each>
	TokenVars     // vars>
	TokenSquash   // squash>
	TokenDistill  // distill>

	// Oracle keywords
	TokenJsonParse    // json_parse
	TokenKeysPresent  // keys_present
	TokenAssert       // assert
	TokenScore        // score
	TokenReject       // reject

	// Type keywords
	TokenStr     // str
	TokenTypeNumber  // number (type keyword)
	TokenBoolean // boolean
	TokenJson    // json
	TokenEnum    // enum
	TokenMol     // mol

	// Graph operations
	TokenOpAdd   // !add
	TokenOpDrop  // !drop
	TokenOpWire  // !wire
	TokenOpCut   // !cut
	TokenOpSplit // !split
	TokenOpMerge // !merge
	TokenOpRefine // !refine
	TokenOpSeed  // !seed

	// Code fence
	TokenCodeFence // ```
	TokenCodeFenceTag // language/type after ```

	// Prompt line content (raw text after section tag)
	TokenPromptText
)

// Token represents a lexed token with position information.
type Token struct {
	Type    TokenType
	Value   string
	Line    int
	Col     int
}

func (t Token) String() string {
	if len(t.Value) > 40 {
		return fmt.Sprintf("%s(%q...@%d:%d)", t.Type, t.Value[:40], t.Line, t.Col)
	}
	return fmt.Sprintf("%s(%q@%d:%d)", t.Type, t.Value, t.Line, t.Col)
}

var tokenNames = map[TokenType]string{
	TokenEOF: "EOF", TokenNewline: "Newline", TokenIndent: "Indent",
	TokenHash: "#", TokenHashSlash: "#/", TokenDoubleHash: "##",
	TokenDoubleHashSlash: "##/", TokenLBrace: "{", TokenRBrace: "}",
	TokenLBracket: "[", TokenRBracket: "]", TokenLParen: "(", TokenRParen: ")",
	TokenComma: ",", TokenColon: ":", TokenDot: ".", TokenPipe: "|",
	TokenArrow: "->", TokenFatArrow: "=>", TokenAt: "@", TokenDash: "-",
	TokenQuestion: "?", TokenEquals: "=", TokenBang: "!", TokenSemicolon: ";",
	TokenDoubleLBrace: "{{", TokenDoubleRBrace: "}}",
	TokenEqEq: "==", TokenNotEq: "!=", TokenLT: "<", TokenGT: ">",
	TokenLTEq: "<=", TokenGTEq: ">=",
	TokenIdent: "Ident", TokenString: "String", TokenNumber: "Number",
	TokenComment: "Comment",
	TokenCodeFence: "```", TokenCodeFenceTag: "CodeFenceTag",
	TokenPromptText: "PromptText",
}

func (t TokenType) String() string {
	if name, ok := tokenNames[t]; ok {
		return name
	}
	return fmt.Sprintf("Token(%d)", int(t))
}
