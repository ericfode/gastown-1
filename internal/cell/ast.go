// Package cell implements the Cell language parser, validator, and pretty-printer.
//
// Cell is a DSL for defining reactive bead computation graphs. A .cell file
// declares cells (computation nodes), their dependencies (refs), oracles
// (cheap output checks), and recipes (named operation sequences).
//
// Syntax overview:
//
//	cell find-bugs {
//	    type: inventory
//	    prompt: """
//	        Find all bugs in {{source-code}}
//	    """
//	    refs: [source-code]
//	    oracle: schema-check
//	}
//
//	recipe enrich(target, source_prompt, refined_prompt) {
//	    src = addCell({ prompt: source_prompt })
//	    addRef(target, src)
//	    refinePrompt(target, refined_prompt)
//	}
package cell

// File represents a parsed .cell file.
type File struct {
	Cells   []*CellDecl
	Recipes []*RecipeDecl
}

// CellDecl represents a cell declaration.
type CellDecl struct {
	Name   string
	Type   string   // e.g. "text", "inventory", "code", "decision", etc.
	Prompt string   // Template string with {{ref}} placeholders.
	Refs   []string // Explicit upstream dependency names.
	Oracle string   // Name of another cell that validates this cell's output.
	Pos    Position // Source location.
}

// RecipeDecl represents a recipe declaration.
type RecipeDecl struct {
	Name   string
	Params []string
	Body   []*Statement
	Pos    Position
}

// Statement represents a statement in a recipe body.
type Statement struct {
	// Exactly one of these is set:
	Assignment *Assignment // e.g. src = addCell(...)
	Call       *Call       // e.g. addRef(target, src)
	Pos        Position
}

// Assignment represents a variable binding: name = call.
type Assignment struct {
	Name string
	Call  *Call
}

// Call represents a function/operation call: name(args...).
type Call struct {
	Name string
	Args []Arg
	Pos  Position
}

// Arg represents an argument to a call.
type Arg struct {
	// Exactly one of these is set:
	Ident  string  // Variable or cell name reference.
	Str    string  // String literal.
	List   []Arg   // List literal: [a, b, c].
	Object []Field // Object literal: { key: value }.
}

// Field represents a key-value pair in an object literal.
type Field struct {
	Key   string
	Value Arg
}

// Position tracks source location for error reporting.
type Position struct {
	File   string
	Line   int
	Column int
}
