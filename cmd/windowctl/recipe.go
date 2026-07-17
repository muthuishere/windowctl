// recipe.go — CLI for the visual recipe system: save/list/run named,
// parameterized sequences of the visual primitives. The visual twin of
// browser-bridge's DOM recipes — see docs/specs/16-recipes.md.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	windowctl "github.com/muthuishere/windowctl"
)

func recipeCmd(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: windowctl recipe (list | save <name> [--file <path>] | run <name> [--var K=V ...] [--confirm] [--json])")
		os.Exit(2)
	}
	switch args[0] {
	case "list":
		recipeListCmd(args[1:])
	case "save":
		recipeSaveCmd(args[1:])
	case "run":
		recipeRunCmd(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "windowctl recipe: unknown subcommand %q (want list, save or run)\n", args[0])
		os.Exit(2)
	}
}

func recipeListCmd(args []string) {
	fs := flag.NewFlagSet("recipe list", flag.ExitOnError)
	asJSON := fs.Bool("json", false, "emit the full recipe set as JSON")
	_ = fs.Parse(args)

	recipes, err := windowctl.ListRecipes()
	if err != nil {
		fmt.Fprintln(os.Stderr, "windowctl:", err)
		os.Exit(1)
	}
	if *asJSON {
		_ = json.NewEncoder(os.Stdout).Encode(recipes)
		return
	}
	for _, r := range recipes {
		scope := strings.TrimSpace(strings.Join([]string{r.Match.App, r.Match.Title}, " "))
		fmt.Printf("%s\t[%s]\t%d steps\tvars=%s\n\t%s\n",
			r.Name, scope, len(r.Steps), strings.Join(r.Vars, ","), r.Description)
	}
}

func recipeSaveCmd(args []string) {
	// <name> is the first positional; Go's flag package stops at the
	// first non-flag arg, so pull the name off before parsing flags.
	if len(args) < 1 || strings.HasPrefix(args[0], "-") {
		fmt.Fprintln(os.Stderr, "windowctl recipe save: <name> is required (before any flags)")
		os.Exit(2)
	}
	name := args[0]
	fs := flag.NewFlagSet("recipe save", flag.ExitOnError)
	file := fs.String("file", "", "recipe body JSON path (default: read stdin)")
	_ = fs.Parse(args[1:])

	var raw []byte
	var err error
	if *file != "" {
		raw, err = os.ReadFile(*file)
	} else {
		raw, err = io.ReadAll(os.Stdin)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "windowctl recipe save:", err)
		os.Exit(1)
	}
	var r windowctl.Recipe
	if err := json.Unmarshal(raw, &r); err != nil {
		fmt.Fprintln(os.Stderr, "windowctl recipe save: body is not a valid recipe JSON:", err)
		os.Exit(1)
	}
	r.Name = name // the positional name is authoritative
	if err := windowctl.SaveRecipe(r); err != nil {
		fmt.Fprintln(os.Stderr, "windowctl:", err)
		os.Exit(1)
	}
	fmt.Printf("saved recipe %q (%d steps)\n", r.Name, len(r.Steps))
}

func recipeRunCmd(args []string) {
	// <name> is the first positional; pull it off before parsing flags
	// (Go's flag package stops at the first non-flag arg).
	if len(args) < 1 || strings.HasPrefix(args[0], "-") {
		fmt.Fprintln(os.Stderr, "windowctl recipe run: <name> is required (before any flags)")
		os.Exit(2)
	}
	name := args[0]
	fs := flag.NewFlagSet("recipe run", flag.ExitOnError)
	confirm := fs.Bool("confirm", false, "allow confirm-gated (submit/publish) steps to fire")
	asJSON := fs.Bool("json", false, "emit the run result (per-step outcomes) as JSON")
	var vars multiVar
	fs.Var(&vars, "var", "recipe variable as K=V (repeatable)")
	_ = fs.Parse(args[1:])

	res, err := windowctl.RunRecipe(name, vars.m, *confirm)
	if *asJSON {
		_ = json.NewEncoder(os.Stdout).Encode(res)
	} else {
		for _, s := range res.Steps {
			fmt.Printf("  %-10s %-8s %s\n", s.Action, s.Status, s.Detail)
		}
		if res.HeldForConfirm {
			fmt.Println("note: a submit step was held back (safe mode) — re-run with --confirm to publish")
		}
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "windowctl:", err)
		os.Exit(1)
	}
}

// multiVar collects repeated --var K=V flags into a map.
type multiVar struct{ m map[string]string }

func (v *multiVar) String() string { return "" }
func (v *multiVar) Set(s string) error {
	if v.m == nil {
		v.m = map[string]string{}
	}
	k, val, ok := strings.Cut(s, "=")
	if !ok || k == "" {
		return fmt.Errorf("--var must be K=V, got %q", s)
	}
	v.m[k] = val
	return nil
}
