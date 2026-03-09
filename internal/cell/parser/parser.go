package parser

import (
	"fmt"
	"strconv"
	"strings"
)

// ParseError represents a parse error with location info.
type ParseError struct {
	Message string
	Pos     Position
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("%d:%d: %s", e.Pos.Line, e.Pos.Col, e.Message)
}

// Parser is a recursive descent parser for Cell source code.
type Parser struct {
	tokens       []Token
	pos          int
	errors       []*ParseError
	pendingWires []*Wire // extra wires from chained wire expressions
}

// Parse tokenizes and parses Cell source code into an AST.
func Parse(source string) (*Program, error) {
	tokens, err := Lex(source)
	if err != nil {
		return nil, err
	}

	p := &Parser{tokens: tokens}
	prog := p.parseProgram()

	if len(p.errors) > 0 {
		var msgs []string
		for _, e := range p.errors {
			msgs = append(msgs, e.Error())
		}
		return prog, fmt.Errorf("parse errors:\n  %s", strings.Join(msgs, "\n  "))
	}

	return prog, nil
}

func (p *Parser) parseProgram() *Program {
	prog := &Program{}

	for !p.atEnd() {
		p.skipNewlines()
		if p.atEnd() {
			break
		}

		switch {
		case p.check(TokenDoubleHash):
			mol := p.parseMolecule()
			if mol != nil {
				prog.Molecules = append(prog.Molecules, mol)
			}
		case p.check(TokenRecipe):
			recipe := p.parseRecipe()
			if recipe != nil {
				prog.Recipes = append(prog.Recipes, recipe)
			}
		case p.check(TokenPromptAt):
			frag := p.parsePromptFragment()
			if frag != nil {
				prog.Fragments = append(prog.Fragments, frag)
			}
		case p.check(TokenHash):
			// Could be oracle_decl or standalone cell
			decl := p.parseTopLevelCell()
			if decl != nil {
				if decl.Type.Name == "oracle" {
					prog.Oracles = append(prog.Oracles, &OracleDecl{
						Name:   decl.Name,
						Oracle: decl.Oracle,
						Pos:    decl.Pos,
					})
				}
			}
		case p.check(TokenInput):
			input := p.parseInputDecl()
			if input != nil {
				prog.Inputs = append(prog.Inputs, input)
			}
		case p.check(TokenComment):
			p.advance() // skip comments
		default:
			p.addError("unexpected token: %s", p.current().Value)
			p.advance()
		}
	}

	return prog
}

func (p *Parser) parseMolecule() *Molecule {
	pos := p.currentPos()
	p.expect(TokenDoubleHash) // ##

	name := p.expectIdent()
	// Accept both ## name { and ## name (braceless) syntax
	if p.check(TokenLBrace) {
		p.advance() // consume optional {
	}
	p.skipNewlines()

	mol := &Molecule{Name: name, Pos: pos}

	for !p.atEnd() && !p.check(TokenDoubleHashSlash) {
		p.skipNewlines()
		if p.atEnd() || p.check(TokenDoubleHashSlash) {
			break
		}

		switch {
		case p.check(TokenComment):
			p.advance()

		case p.check(TokenSquash):
			mol.Squash = p.parseSquashBlock()

		case p.check(TokenInput):
			input := p.parseInputDecl()
			if input != nil {
				mol.Inputs = append(mol.Inputs, input)
			}

		case p.check(TokenPreset):
			preset := p.parsePreset()
			if preset != nil {
				mol.Presets = append(mol.Presets, preset)
			}

		case p.check(TokenPromptAt):
			frag := p.parsePromptFragment()
			if frag != nil {
				mol.Fragments = append(mol.Fragments, frag)
			}

		case p.check(TokenImport):
			imp := p.parseImportDecl()
			if imp != nil {
				mol.Imports = append(mol.Imports, imp)
			}

		case p.check(TokenApply):
			apply := p.parseApplyStmt()
			if apply != nil {
				mol.Applies = append(mol.Applies, apply)
			}

		case p.check(TokenMap):
			mc := p.parseMapCell()
			if mc != nil {
				mol.MapCells = append(mol.MapCells, mc)
			}

		case p.check(TokenReduce):
			rc := p.parseReduceCell()
			if rc != nil {
				mol.ReduceCells = append(mol.ReduceCells, rc)
			}

		case p.check(TokenMeta):
			cell := p.parseMetaCell()
			if cell != nil {
				mol.Cells = append(mol.Cells, cell)
			}

		case p.check(TokenHash):
			cell := p.parseTopLevelCell()
			if cell != nil {
				if cell.Type.Name == "oracle" {
					mol.Oracles = append(mol.Oracles, &OracleDecl{
						Name:   cell.Name,
						Oracle: cell.Oracle,
						Pos:    cell.Pos,
					})
				} else {
					mol.Cells = append(mol.Cells, cell)
				}
			}

		case p.check(TokenIdent), p.check(TokenLBracket):
			// Could be a wire: A -> B, A -> [B, C], [A, B] -> C
			wire := p.parseWire()
			if wire != nil {
				mol.Wires = append(mol.Wires, wire)
			}
			// Collect any chained wires
			if len(p.pendingWires) > 0 {
				mol.Wires = append(mol.Wires, p.pendingWires...)
				p.pendingWires = nil
			}

		default:
			p.addError("unexpected token in molecule body: %s", p.current().Value)
			p.advance()
		}
	}

	p.expect(TokenDoubleHashSlash) // ##/
	return mol
}

func (p *Parser) parseTopLevelCell() *Cell {
	return p.parseCell(false)
}

func (p *Parser) parseMetaCell() *Cell {
	p.expect(TokenMeta) // meta
	cell := p.parseCell(true)
	return cell
}

func (p *Parser) parseCell(isMeta bool) *Cell {
	pos := p.currentPos()
	p.expect(TokenHash) // #

	name := p.parseTemplateName()
	p.expect(TokenColon) // :
	cellType := p.parseCellType()

	cell := &Cell{
		Name:   name,
		Type:   cellType,
		IsMeta: isMeta,
		Pos:    pos,
	}

	p.skipNewlines()
	p.parseCellBody(cell)

	// Expect closing: #/ or meta #/
	if isMeta {
		p.expect(TokenMeta)
	}
	p.expect(TokenHashSlash) // #/

	return cell
}

func (p *Parser) parseCellType() CellType {
	if p.check(TokenMol) {
		p.advance() // mol
		p.expect(TokenLParen)
		name := p.expectIdent()
		p.expect(TokenRParen)
		return CellType{Name: "mol", MolRef: name}
	}

	name := p.expectIdent()
	return CellType{Name: name}
}

func (p *Parser) parseCellBody(cell *Cell) {
	for !p.atEnd() && !p.check(TokenHashSlash) && !p.checkSequence(TokenMeta, TokenHashSlash) {
		p.skipNewlines()
		if p.atEnd() || p.check(TokenHashSlash) || p.checkSequence(TokenMeta, TokenHashSlash) {
			break
		}

		switch {
		case p.check(TokenComment):
			p.advance()

		case p.check(TokenDash):
			ref := p.parseRefDecl()
			if ref != nil {
				cell.Refs = append(cell.Refs, ref)
			}

		case p.check(TokenAt):
			ann := p.parseAnnotation()
			if ann != nil {
				cell.Annotations = append(cell.Annotations, ann)
			}

		case p.checkSectionTag():
			section := p.parsePromptSection()
			if section != nil {
				if section.Tag == "accept" {
					cell.AcceptBlock = &AcceptBlock{
						Lines: section.Lines,
						Pos:   section.Pos,
					}
				} else {
					cell.Prompts = append(cell.Prompts, section)
				}
			}

		case p.check(TokenCodeFence):
			// Peek at tag to distinguish oracle vs script code fences
			if p.isOracleCodeFence() {
				oracle := p.parseOracleBlock()
				if oracle != nil {
					cell.Oracle = oracle
				}
			} else {
				cell.ScriptBody = p.parseScriptCodeFence()
			}

		case p.check(TokenVars):
			vars := p.parseVarsBlock()
			if vars != nil {
				cell.VarsBlock = vars
			}

		case p.check(TokenPromptText):
			// Prompt text that appeared after a code fence inside a prompt section.
			// Append to the last prompt section's lines.
			text := p.current().Value
			p.advance()
			if len(cell.Prompts) > 0 {
				last := cell.Prompts[len(cell.Prompts)-1]
				last.Lines = append(last.Lines, text)
			}

		case p.check(TokenIdent) && p.current().Value == "param" && p.checkParamAssign():
			// param.X = value assignment (used in mol() cells)
			pa := p.parseParamAssign()
			if pa != nil {
				cell.ParamAssigns = append(cell.ParamAssigns, pa)
			}

		default:
			// Unknown — skip
			p.addError("unexpected token in cell body: %s", p.current().Value)
			p.advance()
		}
	}
}

// parseTemplateName parses a name which may contain {{ref}} template parts.
// Examples: "correctness", "{{target}}-draft", "{{target}}-refine-1"
// Stops at :, ->, newline, or other structural tokens.
func (p *Parser) parseTemplateName() string {
	var parts []string
	for !p.atEnd() && !p.check(TokenColon) && !p.check(TokenArrow) &&
		!p.check(TokenNewline) && !p.check(TokenEOF) {
		if p.check(TokenIdent) {
			parts = append(parts, p.current().Value)
			p.advance()
		} else if p.check(TokenDoubleLBrace) {
			p.advance()
			ref := p.expectIdent()
			p.expect(TokenDoubleRBrace)
			parts = append(parts, "{{"+ref+"}}")
		} else if p.check(TokenDash) {
			parts = append(parts, "-")
			p.advance()
		} else if p.check(TokenNumber) {
			parts = append(parts, p.current().Value)
			p.advance()
		} else {
			break
		}
	}
	if len(parts) == 0 {
		p.addError("expected name, got %s (%q)", p.current().Type, p.current().Value)
		return ""
	}
	return strings.Join(parts, "")
}

// checkParamAssign peeks ahead to see if this is param.X = value
func (p *Parser) checkParamAssign() bool {
	// Current is "param", check if next tokens are . ident =
	if p.pos+3 >= len(p.tokens) {
		return false
	}
	return p.tokens[p.pos+1].Type == TokenDot &&
		p.tokens[p.pos+2].Type == TokenIdent &&
		p.tokens[p.pos+3].Type == TokenEquals
}

func (p *Parser) parseParamAssign() *ParamAssign {
	pos := p.currentPos()
	p.advance() // consume "param"
	p.expect(TokenDot)
	name := p.expectIdent()
	p.expect(TokenEquals)
	val := p.parseValue()
	return &ParamAssign{Name: name, Value: val, Pos: pos}
}

func (p *Parser) parseRefDecl() *RefDecl {
	pos := p.currentPos()
	p.expect(TokenDash) // -

	name := p.parseTemplateName()
	ref := &RefDecl{Name: name, Pos: pos}

	// Optional .field or .*
	if p.check(TokenDot) {
		p.advance()
		if p.check(TokenIdent) {
			ref.Field = p.current().Value
			p.advance()
		} else {
			// Could be .* — treat as wildcard
			ref.Field = "*"
		}
	}

	// Optional (or)
	if p.check(TokenLParen) {
		p.advance()
		if p.check(TokenOr) {
			p.advance()
			ref.OrJoin = true
		}
		p.expect(TokenRParen)
	}

	return ref
}

func (p *Parser) parseAnnotation() *Annotation {
	pos := p.currentPos()
	p.expect(TokenAt) // @

	name := p.expectIdent()
	ann := &Annotation{
		Name: name,
		Args: make(map[string]Value),
		Pos:  pos,
	}

	if p.check(TokenLParen) {
		p.advance()
		for !p.atEnd() && !p.check(TokenRParen) {
			key := p.expectIdent()
			p.expect(TokenColon)
			val := p.parseValue()
			ann.Args[key] = val
			if p.check(TokenComma) {
				p.advance()
			}
		}
		p.expect(TokenRParen)
	}

	return ann
}

func (p *Parser) parsePromptSection() *PromptSection {
	pos := p.currentPos()
	tag := p.current().Value
	tag = strings.TrimSuffix(tag, ">")
	p.advance() // consume section tag

	section := &PromptSection{Tag: tag, Pos: pos}

	// Optional guard: ?predicate or ?predicate(args)
	if p.check(TokenQuestion) {
		p.advance()
		section.Guard = p.parseGuard()
	}

	// For each> sections: ident in ref
	if tag == "each" {
		section.Each = p.parseEachSpec()
	}

	// For format> sections: type_name { fields }
	if tag == "format" {
		section.Format = p.parseFormatSpec()
	}

	// Collect prompt lines (indented content until next section or structural token)
	p.skipNewlines()
	section.Lines = p.collectPromptLines()

	return section
}

func (p *Parser) parseGuard() *Guard {
	pos := p.currentPos()
	name := p.expectIdent()
	guard := &Guard{Predicate: name, Pos: pos}

	if p.check(TokenLParen) {
		p.advance()
		for !p.atEnd() && !p.check(TokenRParen) {
			guard.Args = append(guard.Args, p.expectIdent())
			if p.check(TokenComma) {
				p.advance()
			}
		}
		p.expect(TokenRParen)
	}

	return guard
}

func (p *Parser) parseEachSpec() *EachSpec {
	varName := p.expectIdent()
	p.expectKeyword("in")
	// Expect {{ ref }}
	p.expect(TokenDoubleLBrace)
	ref := p.expectIdent()
	// Handle dotted refs like review.*
	if p.check(TokenDot) {
		p.advance()
		suffix := ""
		if p.check(TokenIdent) {
			suffix = p.current().Value
			p.advance()
		} else {
			suffix = "*"
		}
		ref = ref + "." + suffix
	}
	p.expect(TokenDoubleRBrace)
	return &EachSpec{VarName: varName, OverRef: ref}
}

func (p *Parser) parseFormatSpec() *FormatSpec {
	spec := &FormatSpec{}
	if p.check(TokenIdent) || p.check(TokenJson) {
		spec.TypeName = p.current().Value
		p.advance()
	}
	p.skipNewlines()
	if p.check(TokenLBrace) {
		spec.Fields = p.parseFormatFields()
	}
	return spec
}

func (p *Parser) parseFormatFields() []*FormatField {
	p.expect(TokenLBrace)
	p.skipNewlines()
	var fields []*FormatField

	for !p.atEnd() && !p.check(TokenRBrace) {
		p.skipNewlines()
		if p.check(TokenRBrace) {
			break
		}

		name := ""
		if p.check(TokenString) {
			name = p.current().Value
			p.advance()
		} else {
			name = p.expectIdent()
		}
		p.expect(TokenColon)
		ft := p.parseFormatType()
		fields = append(fields, &FormatField{Name: name, Type: ft})

		if p.check(TokenComma) {
			p.advance()
		}
		p.skipNewlines()
	}

	p.expect(TokenRBrace)
	return fields
}

func (p *Parser) parseFormatType() FormatType {
	switch {
	case p.check(TokenStr):
		p.advance()
		return FormatType{Kind: "str"}
	case p.check(TokenTypeNumber):
		p.advance()
		return FormatType{Kind: "number"}
	case p.check(TokenBoolean):
		p.advance()
		return FormatType{Kind: "boolean"}
	case p.check(TokenLBracket):
		p.advance()
		if p.check(TokenIdent) && p.current().Value == "_" {
			p.advance()
			p.expect(TokenRBracket)
			return FormatType{Kind: "wildcard"}
		}
		elem := p.parseFormatType()
		p.expect(TokenRBracket)
		return FormatType{Kind: "array", ElementType: &elem}
	case p.check(TokenLBrace):
		fields := p.parseFormatFields()
		return FormatType{Kind: "object", Fields: fields}
	case p.check(TokenString):
		// Enum: "a" | "b" | "c"
		var values []string
		values = append(values, p.current().Value)
		p.advance()
		for p.check(TokenPipe) {
			p.advance()
			if p.check(TokenString) {
				values = append(values, p.current().Value)
				p.advance()
			}
		}
		return FormatType{Kind: "enum", EnumValues: values}
	default:
		p.addError("expected format type, got: %s", p.current().Value)
		p.advance()
		return FormatType{Kind: "str"}
	}
}

// isOracleCodeFence returns true if the current code fence has an "oracle" tag.
// Must be called when positioned at TokenCodeFence.
func (p *Parser) isOracleCodeFence() bool {
	// Look ahead: TokenCodeFence, then optionally TokenCodeFenceTag
	if p.pos+1 < len(p.tokens) && p.tokens[p.pos+1].Type == TokenCodeFenceTag {
		return p.tokens[p.pos+1].Value == "oracle"
	}
	// No tag — treat as oracle for backward compatibility (bare ``` in oracle position)
	return true
}

// parseScriptCodeFence parses a ```bash ... ``` or ```sh ... ``` block and returns the body.
func (p *Parser) parseScriptCodeFence() string {
	p.expect(TokenCodeFence) // opening ```
	if p.check(TokenCodeFenceTag) {
		p.advance() // consume tag (bash, sh, etc.)
	}
	body := ""
	if p.check(TokenPromptText) {
		body = p.current().Value
		p.advance()
	}
	p.expect(TokenCodeFence) // closing ```
	return body
}

func (p *Parser) parseOracleBlock() *OracleBlock {
	pos := p.currentPos()
	p.expect(TokenCodeFence) // ```

	// Expect "oracle" tag
	if p.check(TokenCodeFenceTag) {
		p.advance() // consume tag
	}

	oracle := &OracleBlock{Pos: pos}

	// The content between fences was lexed as a single PromptText token
	if p.check(TokenPromptText) {
		content := p.current().Value
		p.advance()
		oracle.Statements = parseOracleStatements(content)
	}

	p.expect(TokenCodeFence) // closing ```

	return oracle
}

// parseOracleStatements does a line-by-line parse of oracle content.
// Handles nested for/if blocks by collecting body statements.
func parseOracleStatements(content string) []*OracleStmt {
	lines := strings.Split(content, "\n")
	stmts, _ := parseOracleLines(lines, 0)
	return stmts
}

func parseOracleLines(lines []string, start int) ([]*OracleStmt, int) {
	var stmts []*OracleStmt
	i := start
	for i < len(lines) {
		line := strings.TrimSpace(lines[i])
		if line == "" || strings.HasPrefix(line, "--") {
			i++
			continue
		}
		// End of block
		if line == "}" {
			return stmts, i + 1
		}
		line = strings.TrimSuffix(line, ";")
		line = strings.TrimSpace(line)
		if line == "" {
			i++
			continue
		}

		stmt := &OracleStmt{Expr: line}
		switch {
		case strings.HasPrefix(line, "json_parse"):
			stmt.Kind = "json_parse"
			stmt.Args = extractArgs(line)
		case strings.HasPrefix(line, "keys_present"):
			stmt.Kind = "keys_present"
			stmt.Args = extractArgs(line)
		case strings.HasPrefix(line, "assert"):
			stmt.Kind = "assert"
			stmt.Expr = strings.TrimPrefix(line, "assert ")
		case strings.HasPrefix(line, "reject if"):
			stmt.Kind = "reject"
			stmt.Expr = strings.TrimPrefix(line, "reject if ")
		case strings.HasPrefix(line, "accept if"):
			stmt.Kind = "accept"
			stmt.Expr = strings.TrimPrefix(line, "accept if ")
		case strings.HasPrefix(line, "score("):
			stmt.Kind = "score_if"
			stmt.Expr = line
		case strings.HasPrefix(line, "score "):
			stmt.Kind = "score"
			stmt.Expr = line
		case strings.HasPrefix(line, "for "):
			stmt.Kind = "for"
			// "for VAR in EXPR {" → extract var and expr
			stmt.Expr = strings.TrimSuffix(line, "{")
			stmt.Expr = strings.TrimSpace(stmt.Expr)
			// Parse body until }
			if strings.HasSuffix(line, "{") {
				var nextI int
				stmt.Body, nextI = parseOracleLines(lines, i+1)
				i = nextI
				stmts = append(stmts, stmt)
				continue
			}
		case strings.HasPrefix(line, "if "):
			stmt.Kind = "if"
			stmt.Expr = strings.TrimSuffix(line, "{")
			stmt.Expr = strings.TrimSpace(stmt.Expr)
			if strings.HasSuffix(line, "{") {
				var nextI int
				stmt.Body, nextI = parseOracleLines(lines, i+1)
				i = nextI
				stmts = append(stmts, stmt)
				continue
			}
		default:
			stmt.Kind = "expr"
		}
		stmts = append(stmts, stmt)
		i++
	}
	return stmts, i
}

func extractArgs(line string) []string {
	idx := strings.Index(line, "(")
	if idx < 0 {
		return nil
	}
	end := strings.LastIndex(line, ")")
	if end < 0 {
		return nil
	}
	inner := line[idx+1 : end]
	parts := strings.Split(inner, ",")
	var args []string
	for _, p := range parts {
		args = append(args, strings.TrimSpace(p))
	}
	return args
}

func (p *Parser) parseVarsBlock() *VarsBlock {
	pos := p.currentPos()
	p.expect(TokenVars) // vars>

	vars := &VarsBlock{
		Vars: make(map[string]Value),
		Pos:  pos,
	}

	p.skipNewlines()
	for !p.atEnd() && p.check(TokenIdent) {
		key := p.current().Value
		p.advance()
		p.expect(TokenEquals)
		val := p.parseValue()
		vars.Vars[key] = val
		p.skipNewlines()
	}

	return vars
}

// parseWire parses one or more chained wires: A -> B -> C becomes [A->B, B->C].
// Also handles oracle-gated wires: A -> ? oracle -> B.
// Supports fan-out: A -> [B, C, D] and fan-in: [A, B] -> C.
func (p *Parser) parseWire() *Wire {
	pos := p.currentPos()

	// Parse LHS: either single ident or [ident, ident, ...]
	var from string
	var fromList []string
	if p.check(TokenLBracket) {
		fromList = p.parseIdentList()
	} else {
		from = p.expectIdent()
	}

	p.expect(TokenArrow) // ->

	// Check for oracle-gated wire: A -> ? oracle -> B
	if p.check(TokenQuestion) {
		p.advance()
		oracleGate := p.expectIdent()
		p.expect(TokenArrow) // second ->
		to := p.expectIdent()
		return &Wire{From: from, FromList: fromList, To: to, OracleGate: oracleGate, Pos: pos}
	}

	// Parse RHS: either single ident or [ident, ident, ...]
	var to string
	var toList []string
	if p.check(TokenLBracket) {
		toList = p.parseIdentList()
	} else {
		to = p.expectIdent()
	}

	firstWire := &Wire{From: from, FromList: fromList, To: to, ToList: toList, Pos: pos}

	// Handle chained wires: A -> B -> C -> D
	// Chaining after fan-out/fan-in is not supported (would be ambiguous).
	if len(fromList) > 0 || len(toList) > 0 {
		return firstWire
	}

	p.pendingWires = nil
	prev := to
	for p.check(TokenArrow) {
		p.advance()
		// Could be oracle-gated: -> ? oracle -> next
		if p.check(TokenQuestion) {
			p.advance()
			oracleGate := p.expectIdent()
			p.expect(TokenArrow)
			next := p.expectIdent()
			p.pendingWires = append(p.pendingWires, &Wire{From: prev, To: next, OracleGate: oracleGate, Pos: pos})
			prev = next
		} else if p.check(TokenLBracket) {
			// Chain ending with fan-out: A -> B -> [C, D]
			toList := p.parseIdentList()
			p.pendingWires = append(p.pendingWires, &Wire{From: prev, ToList: toList, Pos: pos})
			break // can't chain further after fan-out
		} else {
			next := p.expectIdent()
			p.pendingWires = append(p.pendingWires, &Wire{From: prev, To: next, Pos: pos})
			prev = next
		}
	}

	return firstWire
}

// parseTemplateIdentList parses [name, name, ...] where names may contain {{param}}.
func (p *Parser) parseTemplateIdentList() []string {
	p.expect(TokenLBracket)
	var list []string
	for !p.atEnd() && !p.check(TokenRBracket) {
		list = append(list, p.parseTemplateName())
		if p.check(TokenComma) {
			p.advance()
		}
	}
	p.expect(TokenRBracket)
	return list
}

// parseIdentList parses [ident, ident, ...] and returns the list of identifiers.
func (p *Parser) parseIdentList() []string {
	p.expect(TokenLBracket)
	var list []string
	for !p.atEnd() && !p.check(TokenRBracket) {
		list = append(list, p.expectIdent())
		if p.check(TokenComma) {
			p.advance()
		}
	}
	p.expect(TokenRBracket)
	return list
}

func (p *Parser) parseMapCell() *MapCell {
	pos := p.currentPos()
	p.expect(TokenMap) // map
	p.expect(TokenHash) // #

	name := p.expectIdent()
	p.expect(TokenColon)
	cellType := p.parseCellType()
	p.expectKeyword("over")

	// Parse the over reference (could be {{ref}})
	overRef := ""
	if p.check(TokenDoubleLBrace) {
		p.advance()
		overRef = p.expectIdent()
		if p.check(TokenDot) {
			p.advance()
			if p.check(TokenIdent) {
				overRef += "." + p.current().Value
				p.advance()
			}
		}
		p.expect(TokenDoubleRBrace)
	} else {
		overRef = p.expectIdent()
	}

	p.expectKeyword("as")
	asIdent := p.expectIdent()

	body := &Cell{Name: name, Type: cellType, Pos: pos}
	p.skipNewlines()
	p.parseCellBody(body)

	p.expect(TokenHashSlash) // #/

	return &MapCell{
		Name:    name,
		Type:    cellType,
		OverRef: overRef,
		AsIdent: asIdent,
		Body:    body,
		Pos:     pos,
	}
}

func (p *Parser) parseReduceCell() *ReduceCell {
	pos := p.currentPos()
	p.expect(TokenReduce) // reduce
	p.expect(TokenHash) // #

	name := p.expectIdent()
	p.expect(TokenColon)
	cellType := p.parseCellType()
	p.expectKeyword("over")

	overRef := ""
	timesN := 0
	if p.check(TokenNumber) {
		// reduce # name : type over N as ... — bounded iteration
		n, err := strconv.Atoi(p.current().Value)
		if err == nil && n > 0 {
			timesN = n
		} else {
			p.addError("reduce 'over N' requires positive integer, got %s", p.current().Value)
		}
		p.advance()
	} else if p.check(TokenDoubleLBrace) {
		p.advance()
		overRef = p.expectIdent()
		if p.check(TokenDot) {
			p.advance()
			if p.check(TokenIdent) {
				overRef += "." + p.current().Value
				p.advance()
			} else {
				// .* wildcard
				overRef += ".*"
			}
		}
		p.expect(TokenDoubleRBrace)
	} else {
		overRef = p.expectIdent()
	}

	p.expectKeyword("as")
	asIdent := p.expectIdent()
	p.expectKeyword("with")
	accIdent := p.expectIdent()
	p.expect(TokenEquals)
	accDefault := p.parseValue()

	// Optional: until(field) for early exit
	untilField := ""
	p.skipNewlines()
	if p.check(TokenIdent) && p.current().Value == "until" {
		p.advance()
		p.expect(TokenLParen)
		untilField = p.expectIdent()
		p.expect(TokenRParen)
	}

	body := &Cell{Name: name, Type: cellType, Pos: pos}
	p.skipNewlines()
	p.parseCellBody(body)

	p.expect(TokenHashSlash) // #/

	return &ReduceCell{
		Name:       name,
		Type:       cellType,
		OverRef:    overRef,
		TimesN:     timesN,
		AsIdent:    asIdent,
		AccIdent:   accIdent,
		AccDefault: accDefault,
		UntilField: untilField,
		Body:       body,
		Pos:        pos,
	}
}

func (p *Parser) parsePreset() *Preset {
	pos := p.currentPos()
	p.expect(TokenPreset) // preset

	name := p.expectIdent()
	p.expect(TokenLBrace)
	p.skipNewlines()

	preset := &Preset{
		Name:   name,
		Fields: make(map[string]Value),
		Pos:    pos,
	}

	for !p.atEnd() && !p.check(TokenRBrace) {
		p.skipNewlines()
		if p.check(TokenRBrace) {
			break
		}
		key := p.expectIdent()
		p.expect(TokenEquals)
		val := p.parseValue()
		preset.Fields[key] = val
		p.skipNewlines()
	}

	p.expect(TokenRBrace)
	return preset
}

func (p *Parser) parseInputDecl() *InputDecl {
	pos := p.currentPos()
	p.expect(TokenInput) // input

	// param.name
	p.expectKeyword("param")
	p.expect(TokenDot)
	paramName := p.expectIdent()

	p.expect(TokenColon)

	// type
	typeName := ""
	if p.check(TokenStr) || p.check(TokenTypeNumber) || p.check(TokenBoolean) || p.check(TokenJson) {
		typeName = p.current().Value
		p.advance()
	} else if p.check(TokenLBracket) {
		// Array type [type]
		p.advance()
		inner := p.expectIdent()
		p.expect(TokenRBracket)
		typeName = "[" + inner + "]"
	} else {
		typeName = p.expectIdent()
	}

	input := &InputDecl{
		ParamName: paramName,
		Type:      typeName,
		Pos:       pos,
	}

	// Modifiers
	for !p.atEnd() && !p.check(TokenNewline) && !p.check(TokenEOF) {
		if p.check(TokenRequired) {
			p.advance()
			input.Required = true
		} else if p.check(TokenRequiredUnless) {
			p.advance()
			p.expect(TokenLParen)
			for !p.atEnd() && !p.check(TokenRParen) {
				// param.name references
				if p.check(TokenIdent) && p.current().Value == "param" {
					p.advance()
					p.expect(TokenDot)
				}
				input.RequiredUnless = append(input.RequiredUnless, p.expectIdent())
				if p.check(TokenComma) {
					p.advance()
				}
			}
			p.expect(TokenRParen)
		} else if p.check(TokenDefault) {
			p.advance()
			p.expect(TokenLParen)
			val := p.parseValue()
			input.Default = &val
			p.expect(TokenRParen)
		} else {
			break
		}
	}

	return input
}

func (p *Parser) parsePromptFragment() *PromptFragment {
	pos := p.currentPos()
	p.expect(TokenPromptAt) // prompt@

	name := p.expectIdent()
	p.skipNewlines()

	frag := &PromptFragment{
		Name:  name,
		Lines: p.collectPromptLinesTopLevel(),
		Pos:   pos,
	}

	return frag
}

func (p *Parser) parseRecipe() *Recipe {
	pos := p.currentPos()
	p.expect(TokenRecipe)

	name := p.expectIdent()
	p.expect(TokenLParen)

	var params []string
	for !p.atEnd() && !p.check(TokenRParen) {
		params = append(params, p.expectIdent())
		if p.check(TokenComma) {
			p.advance()
		}
	}
	p.expect(TokenRParen)
	p.expect(TokenLBrace)
	p.skipNewlines()

	recipe := &Recipe{
		Name:   name,
		Params: params,
		Pos:    pos,
	}

	for !p.atEnd() && !p.check(TokenRBrace) {
		p.skipNewlines()
		if p.check(TokenRBrace) {
			break
		}

		if p.checkBangOp() {
			op := p.parseOperation()
			if op != nil {
				recipe.Operations = append(recipe.Operations, op)
			}
		} else if p.check(TokenComment) {
			p.advance()
		} else {
			p.addError("expected graph operation in recipe, got: %s", p.current().Value)
			p.advance()
		}
	}

	p.expect(TokenRBrace)
	return recipe
}

func (p *Parser) parseOperation() *Operation {
	pos := p.currentPos()
	tok := p.current()
	p.advance()

	op := &Operation{Pos: pos}

	switch tok.Type {
	case TokenOpAdd:
		op.Kind = "add"
		cell := p.parseCell(false)
		op.Cell = cell

	case TokenOpDrop:
		op.Kind = "drop"
		op.Target = p.parseTemplateName()

	case TokenOpWire:
		op.Kind = "wire"
		// Support fan-in: !wire [A, B] -> C and fan-out: !wire A -> [B, C]
		if p.check(TokenLBracket) {
			op.FromList = p.parseTemplateIdentList()
		} else {
			op.From = p.parseTemplateName()
		}
		p.expect(TokenArrow)
		if p.check(TokenLBracket) {
			op.ToList = p.parseTemplateIdentList()
		} else {
			op.To = p.parseTemplateName()
		}

	case TokenOpCut:
		op.Kind = "cut"
		op.From = p.parseTemplateName()
		p.expect(TokenArrow)
		op.To = p.parseTemplateName()

	case TokenOpSplit:
		op.Kind = "split"
		op.Target = p.expectIdent()
		p.expect(TokenFatArrow)
		p.expect(TokenLBracket)
		for !p.atEnd() && !p.check(TokenRBracket) {
			op.Targets = append(op.Targets, p.expectIdent())
			if p.check(TokenComma) {
				p.advance()
			}
		}
		p.expect(TokenRBracket)

	case TokenOpMerge:
		op.Kind = "merge"
		p.expect(TokenLBracket)
		for !p.atEnd() && !p.check(TokenRBracket) {
			op.Sources = append(op.Sources, p.expectIdent())
			if p.check(TokenComma) {
				p.advance()
			}
		}
		p.expect(TokenRBracket)
		p.expect(TokenFatArrow)
		op.Target = p.expectIdent()

	case TokenOpRefine:
		op.Kind = "refine"
		op.Target = p.expectIdent()
		p.expect(TokenLBrace)
		p.skipNewlines()
		op.Lines = p.collectPromptLines()
		p.expect(TokenRBrace)

	case TokenOpSeed:
		op.Kind = "seed"
		op.Target = p.expectIdent()
		p.expect(TokenLBrace)
		val := p.parseValue()
		op.Value = &val
		p.expect(TokenRBrace)
	}

	return op
}

func (p *Parser) parseImportDecl() *ImportDecl {
	pos := p.currentPos()
	p.expect(TokenImport)
	name := p.expectIdent()
	return &ImportDecl{Name: name, Pos: pos}
}

func (p *Parser) parseApplyStmt() *ApplyStmt {
	pos := p.currentPos()
	p.expect(TokenApply)
	name := p.expectIdent()

	p.expect(TokenLParen)
	var args []string
	for !p.atEnd() && !p.check(TokenRParen) {
		if p.check(TokenIdent) {
			args = append(args, p.current().Value)
			p.advance()
		} else {
			// Could be * wildcard
			p.addError("expected argument in apply, got: %s", p.current().Value)
			p.advance()
		}
		if p.check(TokenComma) {
			p.advance()
		}
	}
	p.expect(TokenRParen)

	apply := &ApplyStmt{
		RecipeName: name,
		Args:       args,
		Pos:        pos,
	}

	// Optional where clause
	if p.check(TokenWhere) {
		p.advance()
		apply.Selector = p.parseSelectorExpr()
	}

	return apply
}

func (p *Parser) parseSelectorExpr() *SelectorExpr {
	sel := &SelectorExpr{}
	sel.Predicates = append(sel.Predicates, p.parseSelectorPred())
	for p.check(TokenAnd) {
		p.advance()
		sel.Predicates = append(sel.Predicates, p.parseSelectorPred())
	}
	return sel
}

func (p *Parser) parseSelectorPred() *SelectorPred {
	field := p.expectIdent()
	op := ""
	switch {
	case p.check(TokenEqEq):
		op = "=="
	case p.check(TokenNotEq):
		op = "!="
	case p.check(TokenLT):
		op = "<"
	case p.check(TokenGT):
		op = ">"
	case p.check(TokenLTEq):
		op = "<="
	case p.check(TokenGTEq):
		op = ">="
	case p.check(TokenMatches):
		op = "matches"
	case p.check(TokenContains):
		op = "contains"
	default:
		p.addError("expected comparison operator in selector, got: %s", p.current().Value)
	}
	p.advance()

	value := ""
	if p.check(TokenString) {
		value = p.current().Value
		p.advance()
	} else if p.check(TokenNumber) {
		value = p.current().Value
		p.advance()
	} else {
		value = p.expectIdent()
	}

	return &SelectorPred{Field: field, Op: op, Value: value}
}

func (p *Parser) parseSquashBlock() *SquashBlock {
	pos := p.currentPos()
	p.expect(TokenSquash) // squash>
	p.skipNewlines()

	block := &SquashBlock{Pos: pos}

	// Parse key: value pairs
	for !p.atEnd() && p.check(TokenIdent) {
		key := p.current().Value
		p.advance()
		p.expect(TokenColon)
		switch key {
		case "trigger":
			block.Trigger = p.expectIdent()
		case "template":
			block.Template = p.expectIdent()
		case "include_metrics":
			if p.check(TokenTrue) {
				block.IncludeMetrics = true
				p.advance()
			} else if p.check(TokenFalse) {
				p.advance()
			} else {
				p.addError("expected true/false for include_metrics")
			}
		default:
			p.advance() // skip unknown
		}
		p.skipNewlines()
	}

	return block
}

func (p *Parser) parseValue() Value {
	switch {
	case p.check(TokenString):
		v := Value{Kind: "string", Str: p.current().Value}
		p.advance()
		return v
	case p.check(TokenNumber):
		num, _ := strconv.ParseFloat(p.current().Value, 64)
		v := Value{Kind: "number", Num: num}
		p.advance()
		return v
	case p.check(TokenTrue):
		p.advance()
		return Value{Kind: "bool", Bool: true}
	case p.check(TokenFalse):
		p.advance()
		return Value{Kind: "bool", Bool: false}
	case p.check(TokenNull):
		p.advance()
		return Value{Kind: "null"}
	case p.check(TokenDoubleLBrace):
		// {{ref}} value
		p.advance()
		ref := p.expectIdent()
		if p.check(TokenDot) {
			p.advance()
			if p.check(TokenIdent) {
				ref += "." + p.current().Value
				p.advance()
			}
		}
		p.expect(TokenDoubleRBrace)
		return Value{Kind: "ref", Ref: ref}
	case p.check(TokenLBracket):
		p.advance()
		p.skipNewlines()
		var arr []Value
		for !p.atEnd() && !p.check(TokenRBracket) {
			p.skipNewlines()
			arr = append(arr, p.parseValue())
			if p.check(TokenComma) {
				p.advance()
			}
			p.skipNewlines()
		}
		p.expect(TokenRBracket)
		return Value{Kind: "array", Array: arr}
	case p.check(TokenLBrace):
		p.advance()
		p.skipNewlines()
		obj := make(map[string]Value)
		for !p.atEnd() && !p.check(TokenRBrace) {
			p.skipNewlines()
			key := ""
			if p.check(TokenString) {
				key = p.current().Value
				p.advance()
			} else {
				key = p.expectIdent()
			}
			p.expect(TokenColon)
			obj[key] = p.parseValue()
			if p.check(TokenComma) {
				p.advance()
			}
			p.skipNewlines()
		}
		p.expect(TokenRBrace)
		return Value{Kind: "object", Object: obj}
	default:
		// Try to read as identifier (for enum-like values in presets)
		if p.check(TokenIdent) {
			v := Value{Kind: "string", Str: p.current().Value}
			p.advance()
			return v
		}
		p.addError("expected value, got: %s", p.current().Value)
		p.advance()
		return Value{Kind: "null"}
	}
}

// collectPromptLines collects indented text lines until a structural token is found.
// inCell: true when collecting inside a cell body (## is just markdown, not structural)
func (p *Parser) collectPromptLines() []string {
	return p.collectPromptLinesInner(true)
}

// collectPromptLinesTopLevel collects prompt lines at top level (## IS structural).
func (p *Parser) collectPromptLinesTopLevel() []string {
	return p.collectPromptLinesInner(false)
}

func (p *Parser) collectPromptLinesInner(inCell bool) []string {
	var lines []string
	for !p.atEnd() {
		// Stop at structural tokens
		if p.check(TokenHash) || p.check(TokenHashSlash) ||
			(!inCell && p.check(TokenDoubleHash)) ||
			p.check(TokenDoubleHashSlash) || p.check(TokenCodeFence) ||
			p.checkSectionTag() || p.check(TokenDash) || p.check(TokenAt) ||
			p.check(TokenVars) || p.check(TokenEOF) ||
			p.check(TokenMap) || p.check(TokenReduce) || p.check(TokenMeta) ||
			p.check(TokenPreset) || p.check(TokenInput) || p.check(TokenImport) ||
			p.check(TokenApply) || p.check(TokenRecipe) || p.check(TokenPromptAt) ||
			p.checkSequence(TokenMeta, TokenHashSlash) {
			break
		}

		if p.check(TokenPromptText) {
			lines = append(lines, p.current().Value)
			p.advance()
			continue
		}

		// Collect the rest of tokens on this line as text
		if p.check(TokenNewline) {
			p.advance()
			continue
		}

		// Accumulate remaining tokens as a line of text.
		// Within a line, only code fence and section tags are structural breaks.
		// #/ and ## mid-line are just prompt text (e.g., "delimiters (#/ closers, ## boundaries)")
		var lineTokens []string
		for !p.atEnd() && !p.check(TokenNewline) && !p.check(TokenEOF) {
			if p.check(TokenCodeFence) || p.checkSectionTag() {
				break
			}
			lineTokens = append(lineTokens, p.current().Value)
			p.advance()
		}
		if len(lineTokens) > 0 {
			lines = append(lines, strings.Join(lineTokens, " "))
		}
	}
	return lines
}

// Helper methods

func (p *Parser) current() Token {
	if p.pos >= len(p.tokens) {
		return Token{Type: TokenEOF}
	}
	return p.tokens[p.pos]
}

func (p *Parser) currentPos() Position {
	tok := p.current()
	return Position{Line: tok.Line, Col: tok.Col}
}

func (p *Parser) advance() Token {
	tok := p.current()
	if p.pos < len(p.tokens) {
		p.pos++
	}
	return tok
}

func (p *Parser) check(tt TokenType) bool {
	return p.current().Type == tt
}

func (p *Parser) checkSequence(types ...TokenType) bool {
	for i, tt := range types {
		idx := p.pos + i
		if idx >= len(p.tokens) {
			return false
		}
		// Skip newlines when checking sequence
		actual := p.tokens[idx]
		if actual.Type != tt {
			return false
		}
	}
	return true
}

func (p *Parser) checkSectionTag() bool {
	tt := p.current().Type
	return tt == TokenSystem || tt == TokenContext || tt == TokenUser ||
		tt == TokenThink || tt == TokenExamples || tt == TokenFormat ||
		tt == TokenAccept || tt == TokenEach || tt == TokenSquash ||
		tt == TokenDistill
}

func (p *Parser) checkBangOp() bool {
	tt := p.current().Type
	return tt == TokenOpAdd || tt == TokenOpDrop || tt == TokenOpWire ||
		tt == TokenOpCut || tt == TokenOpSplit || tt == TokenOpMerge ||
		tt == TokenOpRefine || tt == TokenOpSeed
}

func (p *Parser) expect(tt TokenType) Token {
	if p.current().Type == tt {
		return p.advance()
	}
	p.addError("expected %s, got %s (%q)", tt, p.current().Type, p.current().Value)
	return p.current()
}

func (p *Parser) expectIdent() string {
	// Accept keywords as identifiers in position where an ident is expected
	if p.current().Type == TokenIdent {
		return p.advance().Value
	}
	// Many keywords can also serve as identifiers
	switch p.current().Type {
	case TokenStr, TokenTypeNumber, TokenBoolean, TokenJson, TokenEnum,
		TokenMol, TokenMeta, TokenMap, TokenReduce, TokenOver, TokenAs,
		TokenWith, TokenInput, TokenPreset, TokenRecipe, TokenImport,
		TokenApply, TokenWhere, TokenAnd, TokenOr, TokenNot, TokenIn,
		TokenTrue, TokenFalse, TokenNull, TokenIf, TokenFor,
		TokenRequired, TokenRequiredUnless, TokenDefault,
		TokenContains, TokenMatches, TokenTypeof, TokenLen,
		TokenJsonParse, TokenKeysPresent, TokenAssert, TokenScore,
		TokenReject, TokenAccept:
		return p.advance().Value
	}
	p.addError("expected identifier, got %s (%q)", p.current().Type, p.current().Value)
	return ""
}

func (p *Parser) expectKeyword(kw string) {
	if p.current().Value == kw {
		p.advance()
		return
	}
	p.addError("expected keyword %q, got %q", kw, p.current().Value)
}

func (p *Parser) skipNewlines() {
	for p.check(TokenNewline) || p.check(TokenComment) {
		p.advance()
	}
}

func (p *Parser) atEnd() bool {
	return p.current().Type == TokenEOF
}

func (p *Parser) addError(format string, args ...any) {
	pos := p.currentPos()
	p.errors = append(p.errors, &ParseError{
		Message: fmt.Sprintf(format, args...),
		Pos:     pos,
	})
}
