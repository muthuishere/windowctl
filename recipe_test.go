package windowctl

import (
	"testing"
)

// The embedded seeds load and include the proof recipes, each with a
// confirm-gated submit step (safe-by-default).
func TestSeedRecipesLoadAndAreSafeByDefault(t *testing.T) {
	t.Setenv("WINDOWCTL_CONFIG_DIR", t.TempDir())
	t.Setenv("WCTL_RECIPE_STEP_DELAY_MS", "0") // isolate from real ~/.config
	recipes, err := ListRecipes()
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]Recipe{}
	for _, r := range recipes {
		names[r.Name] = r
	}
	yt, ok := names["youtube-comment"]
	if !ok {
		t.Fatal("seed youtube-comment missing")
	}
	// The last step must be a confirm-gated submit.
	last := yt.Steps[len(yt.Steps)-1]
	if !last.Confirm {
		t.Fatalf("youtube-comment final step should be confirm-gated, got %+v", last)
	}
}

// Save round-trips through the config file and overrides a seed by name.
func TestSaveRecipeRoundTripAndOverride(t *testing.T) {
	t.Setenv("WINDOWCTL_CONFIG_DIR", t.TempDir())
	t.Setenv("WCTL_RECIPE_STEP_DELAY_MS", "0")
	r := Recipe{
		Name:  "youtube-comment",
		Steps: []RecipeStep{{Action: "focus"}},
	}
	if err := SaveRecipe(r); err != nil {
		t.Fatal(err)
	}
	got, err := GetRecipe("youtube-comment")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Steps) != 1 || got.Steps[0].Action != "focus" {
		t.Fatalf("user recipe did not override seed: %+v", got.Steps)
	}
}

func TestSaveRecipeRejectsBadAction(t *testing.T) {
	t.Setenv("WINDOWCTL_CONFIG_DIR", t.TempDir())
	t.Setenv("WCTL_RECIPE_STEP_DELAY_MS", "0")
	err := SaveRecipe(Recipe{Name: "x", Steps: []RecipeStep{{Action: "teleport"}}})
	if err == nil {
		t.Fatal("expected validation error for unknown action")
	}
}

// A confirm-gated submit step is skipped without confirm and the result
// flags HeldForConfirm — the safe-by-default guarantee.
func TestRunRecipeHoldsSubmitWithoutConfirm(t *testing.T) {
	t.Setenv("WINDOWCTL_CONFIG_DIR", t.TempDir())
	t.Setenv("WCTL_RECIPE_STEP_DELAY_MS", "0")
	a := newAutomationAdapter()
	a.monitors[0].Focused = true
	a.monitors[1].Focused = false
	a.findMatches = []TextMatch{{Text: "Add a comment", Confidence: 1, ClickX: 5, ClickY: 6}}

	if err := SaveRecipe(Recipe{
		Name:  "fill-and-submit",
		Match: RecipeMatch{App: "Google Chrome"},
		Vars:  []string{"TEXT"},
		Steps: []RecipeStep{
			{Action: "focus"},
			{Action: "find-click", Text: "Add a comment"},
			{Action: "type", Text: "$TEXT", Method: "keystroke"},
			{Action: "find-click", Text: "Comment", Confirm: true},
		},
	}); err != nil {
		t.Fatal(err)
	}

	res, err := runRecipeWith(a, "fill-and-submit", map[string]string{"TEXT": "hi there"}, false)
	if err != nil {
		t.Fatal(err)
	}
	if !res.HeldForConfirm {
		t.Fatal("expected the submit step to be held for confirm")
	}
	if a.typedText != "hi there" {
		t.Fatalf("var substitution/type failed, typed %q", a.typedText)
	}
	// The submit step was skipped, so no second click on "Comment".
	last := res.Steps[len(res.Steps)-1]
	if last.Status != "skipped" {
		t.Fatalf("submit step status = %q, want skipped", last.Status)
	}
}

// With --confirm, the submit step runs (clicks the found button).
func TestRunRecipeConfirmRunsSubmit(t *testing.T) {
	t.Setenv("WINDOWCTL_CONFIG_DIR", t.TempDir())
	t.Setenv("WCTL_RECIPE_STEP_DELAY_MS", "0")
	a := newAutomationAdapter()
	a.monitors[0].Focused = true
	a.monitors[1].Focused = false
	a.findMatches = []TextMatch{{Text: "Comment", Confidence: 1, ClickX: 99, ClickY: 88}}

	_ = SaveRecipe(Recipe{
		Name:  "submit-only",
		Match: RecipeMatch{App: "Google Chrome"},
		Steps: []RecipeStep{{Action: "find-click", Text: "Comment", Confirm: true}},
	})
	res, err := runRecipeWith(a, "submit-only", nil, true)
	if err != nil {
		t.Fatal(err)
	}
	if res.HeldForConfirm {
		t.Fatal("confirm=true should not hold the submit step")
	}
	if a.clickedAt == nil || *a.clickedAt != [2]int{99, 88} {
		t.Fatalf("submit click landed at %v, want [99 88]", a.clickedAt)
	}
}

// find-click falls back to coordinates when OCR finds nothing ("allow
// all ways": vision-first, coordinate fallback).
func TestFindClickCoordinateFallback(t *testing.T) {
	a := newAutomationAdapter()
	a.findMatches = nil // OCR finds nothing
	fx, fy := 300, 400
	detail, err := findClickWith(a, Match{App: "Google Chrome"}, "Nonexistent", &fx, &fy)
	if err != nil {
		t.Fatal(err)
	}
	if a.clickedAt == nil || *a.clickedAt != [2]int{300, 400} {
		t.Fatalf("fallback click at %v, want [300 400]", a.clickedAt)
	}
	if detail == "" {
		t.Fatal("expected a detail string noting the fallback")
	}
}

func TestSubstituteVarsMissingVarErrors(t *testing.T) {
	_, err := substituteVars("hello $TEXT", []string{"TEXT"}, map[string]string{})
	if err == nil {
		t.Fatal("expected error for missing --var TEXT")
	}
	got, err := substituteVars("hello ${TEXT}!", []string{"TEXT"}, map[string]string{"TEXT": "world"})
	if err != nil {
		t.Fatal(err)
	}
	if got != "hello world!" {
		t.Fatalf("substitution = %q", got)
	}
}
