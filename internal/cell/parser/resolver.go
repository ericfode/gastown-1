package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ResolveError is returned when import resolution fails.
type ResolveError struct {
	Import string
	File   string
	Reason string
	Pos    Position
}

func (e *ResolveError) Error() string {
	return fmt.Sprintf("import %q at %d:%d: %s", e.Import, e.Pos.Line, e.Pos.Col, e.Reason)
}

// ResolveResult holds the outcome of resolving imports for a program.
type ResolveResult struct {
	Program *Program
	Errors  []*ResolveError
}

// ResolveOptions configures import resolution.
type ResolveOptions struct {
	// BaseDir is the directory to search for imported .cell files.
	// If empty, uses the current directory.
	BaseDir string

	// SearchPaths are additional directories to search (after BaseDir).
	SearchPaths []string

	// MaxDepth limits import nesting depth (default 10).
	MaxDepth int
}

// Resolve processes all import declarations in a parsed Program, loading
// and merging imported molecules and recipes.
//
// Resolution semantics:
//   - import foo → look for foo.cell in BaseDir, then SearchPaths
//   - Imported molecule cells are merged into the importing molecule (flat)
//   - Imported recipes become available for apply statements
//   - Duplicate cell names across imports are errors
//   - Circular imports are detected and reported
func Resolve(prog *Program, opts ResolveOptions) *ResolveResult {
	if opts.MaxDepth == 0 {
		opts.MaxDepth = 10
	}

	r := &resolver{
		opts:    opts,
		seen:    make(map[string]bool),
		cache:   make(map[string]*Program),
		result:  &ResolveResult{Program: prog},
	}

	r.resolveProgram(prog, 0)
	return r.result
}

type resolver struct {
	opts   ResolveOptions
	seen   map[string]bool     // tracks current resolution chain (cycle detection)
	cache  map[string]*Program // caches parsed files
	result *ResolveResult
}

func (r *resolver) addError(imp *ImportDecl, reason string) {
	r.result.Errors = append(r.result.Errors, &ResolveError{
		Import: imp.Name,
		Pos:    imp.Pos,
		Reason: reason,
	})
}

func (r *resolver) resolveProgram(prog *Program, depth int) {
	for _, mol := range prog.Molecules {
		r.resolveMolecule(mol, depth)
	}

	// After all imports are resolved, apply recipes.
	if depth == 0 {
		r.applyRecipes(prog)
	}
}

func (r *resolver) resolveMolecule(mol *Molecule, depth int) {
	if len(mol.Imports) == 0 {
		return
	}

	if depth > r.opts.MaxDepth {
		for _, imp := range mol.Imports {
			r.addError(imp, fmt.Sprintf("import depth exceeds maximum (%d)", r.opts.MaxDepth))
		}
		return
	}

	for _, imp := range mol.Imports {
		r.resolveImport(mol, imp, depth)
	}
}

func (r *resolver) resolveImport(mol *Molecule, imp *ImportDecl, depth int) {
	// Cycle detection.
	if r.seen[imp.Name] {
		r.addError(imp, fmt.Sprintf("circular import: %s", imp.Name))
		return
	}
	r.seen[imp.Name] = true
	defer func() { delete(r.seen, imp.Name) }()

	// Find and parse the imported file.
	imported, err := r.loadFile(imp.Name)
	if err != nil {
		r.addError(imp, err.Error())
		return
	}

	// Recursively resolve the imported program's imports.
	r.resolveProgram(imported, depth+1)

	// Merge imported content into the importing molecule.
	r.mergeInto(mol, imported, imp)
}

func (r *resolver) loadFile(name string) (*Program, error) {
	// Check cache.
	if prog, ok := r.cache[name]; ok {
		return prog, nil
	}

	// Validate import name: reject path separators and traversal components.
	// Import names must be simple identifiers (may contain hyphens/underscores).
	if strings.ContainsAny(name, `/\`) {
		return nil, fmt.Errorf("invalid import name %q: path separators not allowed", name)
	}
	if name == ".." || name == "." {
		return nil, fmt.Errorf("invalid import name %q: path traversal not allowed", name)
	}

	// Resolve file path.
	filename := name + ".cell"
	// Handle hyphenated names (import foo-bar → foo-bar.cell).
	// Name is used as-is; hyphens are valid in filenames.

	path := r.findFile(filename)
	if path == "" {
		return nil, fmt.Errorf("file not found: %s (searched %s)", filename, r.searchDescription())
	}

	src, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read %s: %v", path, err)
	}

	prog, err := Parse(string(src))
	if err != nil {
		return nil, fmt.Errorf("parse error in %s: %v", path, err)
	}

	r.cache[name] = prog
	return prog, nil
}

func (r *resolver) findFile(filename string) string {
	// Search BaseDir first.
	if r.opts.BaseDir != "" {
		if p, ok := r.safeJoin(r.opts.BaseDir, filename); ok {
			return p
		}
	}

	// Then search paths.
	for _, dir := range r.opts.SearchPaths {
		if p, ok := r.safeJoin(dir, filename); ok {
			return p
		}
	}

	return ""
}

// safeJoin joins dir and filename, then verifies the result stays within dir.
// Returns the resolved path and true if the file exists and is safe.
func (r *resolver) safeJoin(dir, filename string) (string, bool) {
	p := filepath.Join(dir, filename)
	// Resolve to absolute to catch symlink/traversal escapes.
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return "", false
	}
	absP, err := filepath.Abs(p)
	if err != nil {
		return "", false
	}
	// Ensure the resolved path is within the directory.
	if !strings.HasPrefix(absP, absDir+string(filepath.Separator)) {
		return "", false
	}
	if _, err := os.Stat(p); err == nil {
		return p, true
	}
	return "", false
}

func (r *resolver) searchDescription() string {
	var dirs []string
	if r.opts.BaseDir != "" {
		dirs = append(dirs, r.opts.BaseDir)
	}
	dirs = append(dirs, r.opts.SearchPaths...)
	if len(dirs) == 0 {
		return "no search paths configured"
	}
	return strings.Join(dirs, ", ")
}

func (r *resolver) applyRecipes(prog *Program) {
	// Build recipe lookup from all available recipes.
	recipes := make(map[string]*Recipe)
	for _, rec := range prog.Recipes {
		recipes[rec.Name] = rec
	}

	for _, mol := range prog.Molecules {
		for _, apply := range mol.Applies {
			rec, ok := recipes[apply.RecipeName]
			if !ok {
				r.result.Errors = append(r.result.Errors, &ResolveError{
					Import: apply.RecipeName,
					Pos:    apply.Pos,
					Reason: fmt.Sprintf("recipe %q not found", apply.RecipeName),
				})
				continue
			}

			// Determine target args. If there's a where clause, filter
			// cells by selector and use matching names as args.
			args := apply.Args
			if apply.Selector != nil && len(rec.Params) == 1 && len(args) == 0 {
				args = FilterCellsBySelector(mol, apply.Selector)
			}

			// If recipe has 1 param but apply has N args, apply once per arg.
			argSets := [][]string{args}
			if len(rec.Params) == 1 && len(args) > 1 {
				argSets = make([][]string, len(args))
				for i, arg := range args {
					argSets[i] = []string{arg}
				}
			}

			for _, args := range argSets {
				expanded, err := ExpandRecipe(rec, args)
				if err != nil {
					r.result.Errors = append(r.result.Errors, &ResolveError{
						Import: apply.RecipeName,
						Pos:    apply.Pos,
						Reason: err.Error(),
					})
					continue
				}

				if err := ApplyRecipe(mol, expanded); err != nil {
					r.result.Errors = append(r.result.Errors, &ResolveError{
						Import: apply.RecipeName,
						Pos:    apply.Pos,
						Reason: err.Error(),
					})
				}
			}
		}
	}
}

func (r *resolver) mergeInto(mol *Molecule, imported *Program, imp *ImportDecl) {
	// Build a set of existing cell names for collision detection.
	existing := make(map[string]bool)
	for _, c := range mol.Cells {
		existing[c.Name] = true
	}
	for _, mc := range mol.MapCells {
		existing[mc.Name] = true
	}
	for _, rc := range mol.ReduceCells {
		existing[rc.Name] = true
	}

	// Merge molecules: cells from imported molecules become cells of this molecule.
	for _, importedMol := range imported.Molecules {
		for _, c := range importedMol.Cells {
			if existing[c.Name] {
				r.addError(imp, fmt.Sprintf("cell %q already exists (collision with import %q)", c.Name, imp.Name))
				continue
			}
			mol.Cells = append(mol.Cells, c)
			existing[c.Name] = true
		}
		for _, mc := range importedMol.MapCells {
			if existing[mc.Name] {
				r.addError(imp, fmt.Sprintf("map cell %q already exists (collision with import %q)", mc.Name, imp.Name))
				continue
			}
			mol.MapCells = append(mol.MapCells, mc)
			existing[mc.Name] = true
		}
		for _, rc := range importedMol.ReduceCells {
			if existing[rc.Name] {
				r.addError(imp, fmt.Sprintf("reduce cell %q already exists (collision with import %q)", rc.Name, imp.Name))
				continue
			}
			mol.ReduceCells = append(mol.ReduceCells, rc)
			existing[rc.Name] = true
		}

		// Merge wires.
		mol.Wires = append(mol.Wires, importedMol.Wires...)

		// Merge inputs (skip duplicates by param name).
		existingInputs := make(map[string]bool)
		for _, inp := range mol.Inputs {
			existingInputs[inp.ParamName] = true
		}
		for _, inp := range importedMol.Inputs {
			if !existingInputs[inp.ParamName] {
				mol.Inputs = append(mol.Inputs, inp)
				existingInputs[inp.ParamName] = true
			}
		}

		// Merge fragments.
		mol.Fragments = append(mol.Fragments, importedMol.Fragments...)

		// Merge oracles.
		mol.Oracles = append(mol.Oracles, importedMol.Oracles...)
	}

	// Merge recipes into the top-level program.
	r.result.Program.Recipes = append(r.result.Program.Recipes, imported.Recipes...)

	// Merge top-level fragments and oracles.
	r.result.Program.Fragments = append(r.result.Program.Fragments, imported.Fragments...)
	r.result.Program.Oracles = append(r.result.Program.Oracles, imported.Oracles...)
}
