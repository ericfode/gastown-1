package parser

import (
	"fmt"
	"sort"
	"strings"
)

// PrettyPrint formats a parsed Cell AST back into Cell source code.
func PrettyPrint(prog *Program) string {
	var sb strings.Builder
	p := &printer{w: &sb}

	for i, frag := range prog.Fragments {
		if i > 0 {
			p.nl()
		}
		p.printFragment(frag)
	}

	for _, oracle := range prog.Oracles {
		p.nl()
		p.printOracleDecl(oracle)
	}

	for _, input := range prog.Inputs {
		p.printInputDecl(input, 0)
	}

	for i, mol := range prog.Molecules {
		if i > 0 || len(prog.Fragments) > 0 || len(prog.Oracles) > 0 || len(prog.Inputs) > 0 {
			p.nl()
		}
		p.printMolecule(mol)
	}

	for _, recipe := range prog.Recipes {
		p.nl()
		p.printRecipe(recipe)
	}

	return sb.String()
}

type printer struct {
	w *strings.Builder
}

func (p *printer) write(format string, args ...any) {
	fmt.Fprintf(p.w, format, args...)
}

func (p *printer) nl() {
	p.w.WriteString("\n")
}

func (p *printer) indent(level int) {
	p.w.WriteString(strings.Repeat("  ", level))
}

func (p *printer) printMolecule(mol *Molecule) {
	p.write("## %s {\n", mol.Name)

	// Inputs
	for _, input := range mol.Inputs {
		p.printInputDecl(input, 1)
	}
	if len(mol.Inputs) > 0 {
		p.nl()
	}

	// Imports
	for _, imp := range mol.Imports {
		p.indent(1)
		p.write("import %s\n", imp.Name)
	}

	// Squash
	if mol.Squash != nil {
		p.printSquashBlock(mol.Squash, 1)
		p.nl()
	}

	// Fragments
	for _, frag := range mol.Fragments {
		p.printFragment(frag)
		p.nl()
	}

	// Cells
	for _, cell := range mol.Cells {
		p.printCell(cell, 1)
		p.nl()
	}

	// Map cells
	for _, mc := range mol.MapCells {
		p.printMapCell(mc, 1)
		p.nl()
	}

	// Reduce cells
	for _, rc := range mol.ReduceCells {
		p.printReduceCell(rc, 1)
		p.nl()
	}

	// Oracle declarations
	for _, oracle := range mol.Oracles {
		p.printOracleDecl(oracle)
		p.nl()
	}

	// Apply statements
	for _, apply := range mol.Applies {
		p.printApplyStmt(apply, 1)
	}

	// Presets
	for _, preset := range mol.Presets {
		p.printPreset(preset, 1)
		p.nl()
	}

	// Wires
	if len(mol.Wires) > 0 {
		p.indent(1)
		for i, wire := range mol.Wires {
			if i > 0 {
				p.nl()
				p.indent(1)
			}
			p.printWire(wire)
		}
		p.nl()
	}

	p.write("##/\n")
}

func (p *printer) printCell(cell *Cell, level int) {
	p.indent(level)
	if cell.IsMeta {
		p.write("meta ")
	}
	p.write("# %s : %s\n", cell.Name, formatCellType(cell.Type))

	// Refs
	for _, ref := range cell.Refs {
		p.indent(level + 1)
		p.write("- %s", ref.Name)
		if ref.Field != "" {
			p.write(".%s", ref.Field)
		}
		if ref.OrJoin {
			p.write(" (or)")
		}
		p.nl()
	}

	// Annotations
	for _, ann := range cell.Annotations {
		p.indent(level + 1)
		p.write("@ %s", ann.Name)
		if len(ann.Args) > 0 {
			p.write("(")
			keys := sortedKeys(ann.Args)
			for i, k := range keys {
				if i > 0 {
					p.write(", ")
				}
				p.write("%s: %s", k, formatValue(ann.Args[k]))
			}
			p.write(")")
		}
		p.nl()
	}

	// Vars block
	if cell.VarsBlock != nil {
		p.indent(level + 1)
		p.write("vars>\n")
		for k, v := range cell.VarsBlock.Vars {
			p.indent(level + 2)
			p.write("%s = %s\n", k, formatValue(v))
		}
	}

	// Prompt sections
	for _, section := range cell.Prompts {
		p.printPromptSection(section, level+1)
	}

	// Oracle block
	if cell.Oracle != nil {
		p.printOracleBlock(cell.Oracle, level+1)
	}

	// Accept block
	if cell.AcceptBlock != nil {
		p.indent(level + 1)
		p.write("accept>\n")
		for _, line := range cell.AcceptBlock.Lines {
			p.indent(level + 2)
			p.write("%s\n", line)
		}
	}

	p.indent(level)
	if cell.IsMeta {
		p.write("meta ")
	}
	p.write("#/\n")
}

func (p *printer) printMapCell(mc *MapCell, level int) {
	p.indent(level)
	p.write("map # %s : %s over {{%s}} as %s\n",
		mc.Name, formatCellType(mc.Type), mc.OverRef, mc.AsIdent)

	if mc.Body != nil {
		// Print body contents without cell delimiters
		for _, ref := range mc.Body.Refs {
			p.indent(level + 1)
			p.write("- %s", ref.Name)
			if ref.Field != "" {
				p.write(".%s", ref.Field)
			}
			p.nl()
		}
		for _, ann := range mc.Body.Annotations {
			p.indent(level + 1)
			p.write("@ %s", ann.Name)
			if len(ann.Args) > 0 {
				p.write("(")
				keys := sortedKeys(ann.Args)
				for i, k := range keys {
					if i > 0 {
						p.write(", ")
					}
					p.write("%s: %s", k, formatValue(ann.Args[k]))
				}
				p.write(")")
			}
			p.nl()
		}
		for _, section := range mc.Body.Prompts {
			p.printPromptSection(section, level+1)
		}
		if mc.Body.Oracle != nil {
			p.printOracleBlock(mc.Body.Oracle, level+1)
		}
	}

	p.indent(level)
	p.write("#/\n")
}

func (p *printer) printReduceCell(rc *ReduceCell, level int) {
	p.indent(level)
	if rc.TimesN > 0 {
		p.write("reduce # %s : %s over %d as %s with %s = %s",
			rc.Name, formatCellType(rc.Type), rc.TimesN, rc.AsIdent,
			rc.AccIdent, formatValue(rc.AccDefault))
	} else {
		p.write("reduce # %s : %s over {{%s}} as %s with %s = %s",
			rc.Name, formatCellType(rc.Type), rc.OverRef, rc.AsIdent,
			rc.AccIdent, formatValue(rc.AccDefault))
	}
	if rc.UntilField != "" {
		p.write(" until(%s)", rc.UntilField)
	}
	p.write("\n")

	if rc.Body != nil {
		for _, section := range rc.Body.Prompts {
			p.printPromptSection(section, level+1)
		}
	}

	p.indent(level)
	p.write("#/\n")
}

func (p *printer) printPromptSection(section *PromptSection, level int) {
	p.indent(level)
	p.write("%s>", section.Tag)
	if section.Guard != nil {
		p.write(" ?%s", section.Guard.Predicate)
		if len(section.Guard.Args) > 0 {
			p.write("(%s)", strings.Join(section.Guard.Args, ", "))
		}
	}
	if section.Each != nil {
		p.write(" %s in {{%s}}", section.Each.VarName, section.Each.OverRef)
	}
	if section.Format != nil {
		if section.Format.TypeName != "" {
			p.write(" %s", section.Format.TypeName)
		}
	}
	p.nl()
	for _, line := range section.Lines {
		p.indent(level + 1)
		p.write("%s\n", line)
	}
	if section.Format != nil && len(section.Format.Fields) > 0 {
		p.indent(level + 1)
		p.printFormatFields(section.Format.Fields)
		p.nl()
	}
}

func (p *printer) printFormatFields(fields []*FormatField) {
	p.write("{ ")
	for i, f := range fields {
		if i > 0 {
			p.write(", ")
		}
		p.write("%q: %s", f.Name, formatFormatType(f.Type))
	}
	p.write(" }")
}

func formatFormatType(ft FormatType) string {
	switch ft.Kind {
	case "str":
		return "str"
	case "number":
		return "number"
	case "boolean":
		return "boolean"
	case "array":
		if ft.ElementType != nil {
			return "[" + formatFormatType(*ft.ElementType) + "]"
		}
		return "[_]"
	case "wildcard":
		return "[_]"
	case "object":
		var parts []string
		for _, f := range ft.Fields {
			parts = append(parts, fmt.Sprintf("%q: %s", f.Name, formatFormatType(f.Type)))
		}
		return "{ " + strings.Join(parts, ", ") + " }"
	case "enum":
		var quoted []string
		for _, v := range ft.EnumValues {
			quoted = append(quoted, fmt.Sprintf("%q", v))
		}
		return strings.Join(quoted, " | ")
	default:
		return ft.Kind
	}
}

func (p *printer) printOracleBlock(oracle *OracleBlock, level int) {
	p.indent(level)
	p.write("``` oracle\n")
	for _, stmt := range oracle.Statements {
		p.indent(level)
		p.write("%s;\n", stmt.Expr)
	}
	p.indent(level)
	p.write("```\n")
}

func (p *printer) printOracleDecl(oracle *OracleDecl) {
	p.write("# %s : oracle\n", oracle.Name)
	if oracle.Oracle != nil {
		p.printOracleBlock(oracle.Oracle, 1)
	}
	p.write("#/\n")
}

func (p *printer) printFragment(frag *PromptFragment) {
	p.write("prompt@ %s\n", frag.Name)
	for _, line := range frag.Lines {
		p.write("  %s\n", line)
	}
}

func (p *printer) printInputDecl(input *InputDecl, level int) {
	p.indent(level)
	p.write("input param.%s : %s", input.ParamName, input.Type)
	if input.Required {
		p.write(" required")
	}
	if len(input.RequiredUnless) > 0 {
		p.write(" required_unless(")
		for i, ref := range input.RequiredUnless {
			if i > 0 {
				p.write(", ")
			}
			p.write("param.%s", ref)
		}
		p.write(")")
	}
	if input.Default != nil {
		p.write(" default(%s)", formatValue(*input.Default))
	}
	p.nl()
}

func (p *printer) printPreset(preset *Preset, level int) {
	p.indent(level)
	p.write("preset %s {\n", preset.Name)
	for k, v := range preset.Fields {
		p.indent(level + 1)
		p.write("%s = %s\n", k, formatValue(v))
	}
	p.indent(level)
	p.write("}\n")
}

func (p *printer) printWire(wire *Wire) {
	p.write("%s -> ", wire.From)
	if wire.OracleGate != "" {
		p.write("? %s -> ", wire.OracleGate)
	}
	p.write("%s", wire.To)
}

func (p *printer) printRecipe(recipe *Recipe) {
	p.write("recipe %s(%s) {\n", recipe.Name, strings.Join(recipe.Params, ", "))
	for _, op := range recipe.Operations {
		p.indent(1)
		p.printOperation(op)
	}
	p.write("}\n")
}

func (p *printer) printOperation(op *Operation) {
	switch op.Kind {
	case "add":
		p.write("!add ")
		if op.Cell != nil {
			p.printCell(op.Cell, 0)
		}
	case "drop":
		p.write("!drop %s\n", op.Target)
	case "wire":
		p.write("!wire %s -> %s\n", op.From, op.To)
	case "cut":
		p.write("!cut %s -> %s\n", op.From, op.To)
	case "split":
		p.write("!split %s => [%s]\n", op.Target, strings.Join(op.Targets, ", "))
	case "merge":
		p.write("!merge [%s] => %s\n", strings.Join(op.Sources, ", "), op.Target)
	case "refine":
		p.write("!refine %s {\n", op.Target)
		for _, line := range op.Lines {
			p.indent(2)
			p.write("%s\n", line)
		}
		p.indent(1)
		p.write("}\n")
	case "seed":
		p.write("!seed %s { ", op.Target)
		if op.Value != nil {
			p.write("%s", formatValue(*op.Value))
		}
		p.write(" }\n")
	}
}

func (p *printer) printApplyStmt(apply *ApplyStmt, level int) {
	p.indent(level)
	p.write("apply %s(%s)", apply.RecipeName, strings.Join(apply.Args, ", "))
	if apply.Selector != nil {
		p.write(" where ")
		for i, pred := range apply.Selector.Predicates {
			if i > 0 {
				p.write(" and ")
			}
			p.write("%s %s %s", pred.Field, pred.Op, pred.Value)
		}
	}
	p.nl()
}

func (p *printer) printSquashBlock(sq *SquashBlock, level int) {
	p.indent(level)
	p.write("squash>\n")
	if sq.Trigger != "" {
		p.indent(level + 1)
		p.write("trigger: %s\n", sq.Trigger)
	}
	if sq.Template != "" {
		p.indent(level + 1)
		p.write("template: %s\n", sq.Template)
	}
	if sq.IncludeMetrics {
		p.indent(level + 1)
		p.write("include_metrics: true\n")
	}
}

func formatCellType(ct CellType) string {
	if ct.MolRef != "" {
		return "mol(" + ct.MolRef + ")"
	}
	return ct.Name
}

func formatValue(v Value) string {
	switch v.Kind {
	case "string":
		return fmt.Sprintf("%q", v.Str)
	case "number":
		if v.Num == float64(int(v.Num)) {
			return fmt.Sprintf("%d", int(v.Num))
		}
		return fmt.Sprintf("%g", v.Num)
	case "bool":
		if v.Bool {
			return "true"
		}
		return "false"
	case "null":
		return "null"
	case "ref":
		return "{{" + v.Ref + "}}"
	case "array":
		var items []string
		for _, item := range v.Array {
			items = append(items, formatValue(item))
		}
		return "[" + strings.Join(items, ", ") + "]"
	case "object":
		var pairs []string
		for k, val := range v.Object {
			pairs = append(pairs, fmt.Sprintf("%s: %s", k, formatValue(val)))
		}
		return "{ " + strings.Join(pairs, ", ") + " }"
	default:
		return v.Str
	}
}

func sortedKeys(m map[string]Value) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
