package parser

// Program is the root AST node — a Cell source file.
type Program struct {
	Molecules  []*Molecule
	Recipes    []*Recipe
	Fragments  []*PromptFragment
	Oracles    []*OracleDecl
	Inputs     []*InputDecl
}

// Molecule is a top-level ## name { ... ##/ construct.
type Molecule struct {
	Name       string
	Cells      []*Cell
	MapCells   []*MapCell
	ReduceCells []*ReduceCell
	Wires      []*Wire
	Presets    []*Preset
	Inputs     []*InputDecl
	Fragments  []*PromptFragment
	Oracles    []*OracleDecl
	Imports    []*ImportDecl
	Applies    []*ApplyStmt
	Squash     *SquashBlock
	Pos        Position
}

// Cell is a # name : type ... #/ construct.
type Cell struct {
	Name        string
	Type        CellType
	IsMeta      bool
	Refs        []*RefDecl
	Annotations []*Annotation
	Prompts     []*PromptSection
	Oracle      *OracleBlock
	AcceptBlock *AcceptBlock
	VarsBlock   *VarsBlock
	ScriptBody  string         // bash/sh code fence body for script cells
	ParamAssigns []*ParamAssign // param.X = value assignments (for mol() cells)
	Pos         Position
}

// ParamAssign is a param.name = value assignment inside a mol() cell body.
type ParamAssign struct {
	Name  string
	Value Value
	Pos   Position
}

// CellType represents the type of a cell.
type CellType struct {
	Name    string // "llm", "script", "oracle", "decision", "meta", etc.
	MolRef  string // for mol(name) — the referenced molecule name
}

// MapCell is a map # name : type over ref as ident ... #/ construct.
type MapCell struct {
	Name     string
	Type     CellType
	OverRef  string
	AsIdent  string
	Body     *Cell // reuses Cell body structure
	Pos      Position
}

// ReduceCell is a reduce # name : type over ref as ident with acc = val ... #/ construct.
// When TimesN > 0, the reduce iterates TimesN times (bounded loop) instead of over a ref.
// UntilField enables early exit: iteration stops when acc.UntilField is truthy.
type ReduceCell struct {
	Name       string
	Type       CellType
	OverRef    string   // cell ref to iterate over (empty when TimesN > 0)
	TimesN     int      // bounded iteration count (0 = use OverRef)
	AsIdent    string
	AccIdent   string
	AccDefault Value
	UntilField string   // early exit: stop when acc[UntilField] is truthy (empty = no early exit)
	Body       *Cell
	Pos        Position
}

// RefDecl is a - name or - name.field or - name (or) dependency declaration.
type RefDecl struct {
	Name    string
	Field   string // optional .field
	OrJoin  bool   // (or) modifier
	Pos     Position
}

// Annotation is an @ name(key: val, ...) annotation.
type Annotation struct {
	Name string
	Args map[string]Value
	Pos  Position
}

// PromptSection is a section_tag> [?guard] prompt_lines construct.
type PromptSection struct {
	Tag    string // "system", "context", "user", "think", "examples", "format", "each", "accept"
	Guard  *Guard
	Lines  []string
	Format *FormatSpec // for format> sections
	Each   *EachSpec   // for each> sections
	Pos    Position
}

// Guard is a ?predicate or ?predicate(refs) conditional.
type Guard struct {
	Predicate string
	Args      []string
	Pos       Position
}

// FormatSpec describes a format> section's type structure.
type FormatSpec struct {
	TypeName string
	Fields   []*FormatField
}

// FormatField is a field in a format spec.
type FormatField struct {
	Name string
	Type FormatType
}

// FormatType represents a type in a format spec.
type FormatType struct {
	Kind       string // "str", "number", "boolean", "array", "object", "enum", "wildcard"
	ElementType *FormatType // for arrays
	Fields     []*FormatField // for objects
	EnumValues []string // for enums
}

// EachSpec describes an each> section.
type EachSpec struct {
	VarName string
	OverRef string
}

// OracleBlock is a ``` oracle ... ``` construct.
type OracleBlock struct {
	Statements []*OracleStmt
	Pos        Position
}

// OracleStmt is a statement within an oracle block.
type OracleStmt struct {
	Kind      string // "json_parse", "keys_present", "assert", "for", "if", "score", "reject", "accept", "score_if"
	Args      []string
	Expr      string    // for assert, reject if, accept if, score() if
	Body      []*OracleStmt // for if/for blocks
	ScoreClauses []*ScoreClause // for score blocks
	Pos       Position
}

// ScoreClause is a +N if expr clause in a score block.
type ScoreClause struct {
	Weight float64
	Expr   string
}

// AcceptBlock is an accept> prompt_lines construct.
type AcceptBlock struct {
	Lines []string
	Pos   Position
}

// VarsBlock is a vars> key = value ... construct.
type VarsBlock struct {
	Vars map[string]Value
	Pos  Position
}

// Wire is an A -> B or A -> ? oracle -> B construct.
// Fan-out: A -> [B, C, D] creates wires A->B, A->C, A->D.
// Fan-in: [A, B] -> C creates wires A->C, B->C.
type Wire struct {
	From       string
	To         string
	FromList   []string // fan-in: multiple sources (if set, From is empty)
	ToList     []string // fan-out: multiple targets (if set, To is empty)
	OracleGate string   // optional oracle in the middle
	Pos        Position
}

// Preset is a preset name { key = val ... } construct.
type Preset struct {
	Name   string
	Fields map[string]Value
	Pos    Position
}

// InputDecl is an input param.name : type modifiers construct.
type InputDecl struct {
	ParamName  string
	Type       string
	Required   bool
	RequiredUnless []string
	Default    *Value
	Pos        Position
}

// PromptFragment is a prompt@ name lines construct.
type PromptFragment struct {
	Name  string
	Lines []string
	Pos   Position
}

// OracleDecl is a standalone # name : oracle ... #/ construct.
type OracleDecl struct {
	Name   string
	Oracle *OracleBlock
	Pos    Position
}

// Recipe is a recipe name(params) { operations } construct.
type Recipe struct {
	Name       string
	Params     []string
	Operations []*Operation
	Pos        Position
}

// Operation is a graph operation (!add, !drop, !wire, etc.).
type Operation struct {
	Kind    string // "add", "drop", "wire", "cut", "split", "merge", "refine", "seed"
	Cell    *Cell  // for !add
	Target  string // for !drop, !refine, !seed
	From     string   // for !wire, !cut
	To       string   // for !wire, !cut
	FromList []string // for !wire fan-in
	ToList   []string // for !wire fan-out
	Sources  []string // for !merge (sources)
	Targets  []string // for !split (targets)
	Lines   []string // for !refine (prompt lines)
	Value   *Value   // for !seed
	Pos     Position
}

// ImportDecl is an import name construct.
type ImportDecl struct {
	Name string
	Pos  Position
}

// ApplyStmt is an apply name(args) [where selectors] construct.
type ApplyStmt struct {
	RecipeName string
	Args       []string
	Selector   *SelectorExpr
	Pos        Position
}

// SelectorExpr is a where clause with predicates.
type SelectorExpr struct {
	Predicates []*SelectorPred
}

// SelectorPred is a single predicate in a selector (e.g., type == llm).
type SelectorPred struct {
	Field string // "type", "depth", "tag", "name"
	Op    string // "==", "!=", "<", ">", "<=", ">="
	Value string
}

// SquashBlock is a squash> configuration block.
type SquashBlock struct {
	Trigger        string
	Template       string
	IncludeMetrics bool
	Pos            Position
}

// Value represents a literal value (string, number, bool, null, array, object).
type Value struct {
	Kind     string // "string", "number", "bool", "null", "array", "object", "ref"
	Str      string
	Num      float64
	Bool     bool
	Array    []Value
	Object   map[string]Value
	Ref      string // for {{ref}} values
}

// Position tracks source location.
type Position struct {
	Line int
	Col  int
}
