package cell

import (
	"fmt"
	"strings"
)

// PrettyPrint formats a parsed Cell file as canonical Cell source code.
func PrettyPrint(f *File) string {
	var sb strings.Builder

	for i, c := range f.Cells {
		if i > 0 {
			sb.WriteString("\n")
		}
		printCell(&sb, c)
	}

	for i, r := range f.Recipes {
		if i > 0 || len(f.Cells) > 0 {
			sb.WriteString("\n")
		}
		printRecipe(&sb, r)
	}

	return sb.String()
}

func printCell(sb *strings.Builder, c *CellDecl) {
	fmt.Fprintf(sb, "cell %s {\n", c.Name)

	if c.Type != "" {
		fmt.Fprintf(sb, "    type: %s\n", c.Type)
	}

	if c.Prompt != "" {
		if strings.Contains(c.Prompt, "\n") {
			fmt.Fprintf(sb, "    prompt: \"\"\"\n")
			for _, line := range strings.Split(c.Prompt, "\n") {
				fmt.Fprintf(sb, "        %s\n", line)
			}
			fmt.Fprintf(sb, "    \"\"\"\n")
		} else {
			fmt.Fprintf(sb, "    prompt: %q\n", c.Prompt)
		}
	}

	if len(c.Refs) > 0 {
		fmt.Fprintf(sb, "    refs: [%s]\n", strings.Join(c.Refs, ", "))
	}

	if c.Oracle != "" {
		fmt.Fprintf(sb, "    oracle: %s\n", c.Oracle)
	}

	sb.WriteString("}\n")
}

func printRecipe(sb *strings.Builder, r *RecipeDecl) {
	fmt.Fprintf(sb, "recipe %s(%s) {\n", r.Name, strings.Join(r.Params, ", "))

	for _, stmt := range r.Body {
		sb.WriteString("    ")
		if stmt.Assignment != nil {
			fmt.Fprintf(sb, "%s = ", stmt.Assignment.Name)
			printCall(sb, stmt.Assignment.Call)
		} else if stmt.Call != nil {
			printCall(sb, stmt.Call)
		}
		sb.WriteString("\n")
	}

	sb.WriteString("}\n")
}

func printCall(sb *strings.Builder, c *Call) {
	fmt.Fprintf(sb, "%s(", c.Name)
	for i, arg := range c.Args {
		if i > 0 {
			sb.WriteString(", ")
		}
		printArg(sb, arg)
	}
	sb.WriteString(")")
}

func printArg(sb *strings.Builder, a Arg) {
	switch {
	case a.Ident != "":
		sb.WriteString(a.Ident)
	case a.Str != "":
		fmt.Fprintf(sb, "%q", a.Str)
	case a.List != nil:
		sb.WriteString("[")
		for i, item := range a.List {
			if i > 0 {
				sb.WriteString(", ")
			}
			printArg(sb, item)
		}
		sb.WriteString("]")
	case a.Object != nil:
		sb.WriteString("{ ")
		for i, field := range a.Object {
			if i > 0 {
				sb.WriteString(", ")
			}
			fmt.Fprintf(sb, "%s: ", field.Key)
			printArg(sb, field.Value)
		}
		sb.WriteString(" }")
	}
}
