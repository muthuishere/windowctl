package windowctl

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// recipes.seed.json ships a few real, ready-to-run recipes (youtube-
// comment, x-compose) so the visual write rail is useful out of the box.
// User recipes in the config file override seeds by name.
//
//go:embed recipes.seed.json
var seedRecipesJSON []byte

// A Recipe is a named, parameterized sequence of the visual primitives
// (find→click, type, key, scroll, drag, clipboard-paste, wait-for-text),
// scoped to a window Match, with $VAR substitution. It is the visual
// twin of a browser-bridge DOM recipe: browser-bridge reads/writes the
// DOM, windowctl drives ANY app's UI like a human — so a flow captured
// here replays against apps that have no DOM or whose DOM (DraftJS,
// canvas, native) resists scripting.
type Recipe struct {
	Name        string       `json:"name"`
	Description string       `json:"description,omitempty"`
	Match       RecipeMatch  `json:"match,omitempty"`
	Vars        []string     `json:"vars,omitempty"`
	Steps       []RecipeStep `json:"steps"`
}

// RecipeMatch scopes a recipe (and, unless a step overrides it, each
// step) to a window. Same semantics as everywhere else: Title is a
// case-insensitive substring, App a case-insensitive exact name.
type RecipeMatch struct {
	App   string `json:"app,omitempty"`
	Title string `json:"title,omitempty"`
}

// RecipeStep is one action. Action selects the primitive; the other
// fields parameterize it. Text supports $VAR / ${VAR} substitution.
//
// Actions:
//   - focus                 — focus the (step or recipe) matched window
//   - launch                — launch App (or the match's app)
//   - wait-text             — poll find --text until Text appears (TimeoutMS)
//   - find-click            — find --text Text, click the top match;
//     falls back to (X,Y) coords when OCR finds nothing
//   - click                 — click at (X,Y)
//   - type                  — type Text; Method keystroke|paste|auto
//   - paste                 — SetClipboard(Text) then cmd+v (explicit)
//   - key                   — press Combo
//   - scroll                — wheel DX/DY at (X,Y) or the cursor
//   - drag                  — drag (X,Y)→(ToX,ToY), Button
//
// Confirm gates destructive/outward steps (the final submit/post click):
// such a step is SKIPPED unless the recipe is run with confirm=true, so
// recipes are safe-by-default — they fill a box but never publish on
// their own. Optional steps that fail are skipped instead of aborting.
type RecipeStep struct {
	Action    string `json:"action"`
	Text      string `json:"text,omitempty"`
	Method    string `json:"method,omitempty"`
	Combo     string `json:"combo,omitempty"`
	App       string `json:"app,omitempty"`
	Title     string `json:"title,omitempty"`
	X         *int   `json:"x,omitempty"`
	Y         *int   `json:"y,omitempty"`
	ToX       *int   `json:"to_x,omitempty"`
	ToY       *int   `json:"to_y,omitempty"`
	DX        int    `json:"dx,omitempty"`
	DY        int    `json:"dy,omitempty"`
	Button    string `json:"button,omitempty"`
	TimeoutMS int    `json:"timeout_ms,omitempty"`
	Confirm   bool   `json:"confirm,omitempty"`
	Optional  bool   `json:"optional,omitempty"`
	Note      string `json:"note,omitempty"`
}

// recipeFile is the on-disk shape (both the seed and the user config).
type recipeFile struct {
	Recipes []Recipe `json:"recipes"`
}

// StepOutcome records what happened to one step, for the run summary /
// receipt. Status is "ok", "skipped" (confirm-gated or optional-fail),
// or "failed".
type StepOutcome struct {
	Action string `json:"action"`
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`
}

// RecipeResult is the outcome of RunRecipe: one StepOutcome per step, in
// order, plus whether a confirm-gated (submit) step was held back.
type RecipeResult struct {
	Name           string        `json:"name"`
	Steps          []StepOutcome `json:"steps"`
	HeldForConfirm bool          `json:"held_for_confirm"`
}

// recipesConfigPath returns the user recipes file path. Override the
// directory with WINDOWCTL_CONFIG_DIR (used by tests and by callers who
// keep runtime state outside ~/.config).
func recipesConfigPath() (string, error) {
	dir := os.Getenv("WINDOWCTL_CONFIG_DIR")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		dir = filepath.Join(home, ".config", "windowctl")
	}
	return filepath.Join(dir, "recipes.json"), nil
}

// ListRecipes returns the merged recipe set (embedded seeds overlaid by
// user recipes of the same name), sorted by name.
func ListRecipes() ([]Recipe, error) {
	merged, err := loadMergedRecipes()
	if err != nil {
		return nil, err
	}
	out := make([]Recipe, 0, len(merged))
	for _, r := range merged {
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// GetRecipe returns one recipe by name from the merged set.
func GetRecipe(name string) (Recipe, error) {
	merged, err := loadMergedRecipes()
	if err != nil {
		return Recipe{}, err
	}
	r, ok := merged[name]
	if !ok {
		return Recipe{}, fmt.Errorf("recipe %q not found (try `windowctl recipe list`)", name)
	}
	return r, nil
}

// SaveRecipe validates and persists a recipe to the user config file,
// replacing any existing user recipe of the same name.
func SaveRecipe(r Recipe) error {
	if err := validateRecipe(r); err != nil {
		return err
	}
	path, err := recipesConfigPath()
	if err != nil {
		return err
	}
	var file recipeFile
	if b, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(b, &file)
	}
	replaced := false
	for i := range file.Recipes {
		if file.Recipes[i].Name == r.Name {
			file.Recipes[i] = r
			replaced = true
			break
		}
	}
	if !replaced {
		file.Recipes = append(file.Recipes, r)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

func validateRecipe(r Recipe) error {
	if strings.TrimSpace(r.Name) == "" {
		return errors.New("recipe: name is required")
	}
	if len(r.Steps) == 0 {
		return errors.New("recipe: at least one step is required")
	}
	for i, s := range r.Steps {
		switch s.Action {
		case "focus", "launch", "wait-text", "find-click", "click", "type", "paste", "key", "scroll", "drag":
		default:
			return fmt.Errorf("recipe: step %d has unknown action %q", i+1, s.Action)
		}
	}
	return nil
}

func loadMergedRecipes() (map[string]Recipe, error) {
	merged := map[string]Recipe{}
	var seed recipeFile
	if err := json.Unmarshal(seedRecipesJSON, &seed); err != nil {
		return nil, fmt.Errorf("recipe: corrupt embedded seeds: %w", err)
	}
	for _, r := range seed.Recipes {
		merged[r.Name] = r
	}
	path, err := recipesConfigPath()
	if err != nil {
		return nil, err
	}
	if b, err := os.ReadFile(path); err == nil {
		var user recipeFile
		if err := json.Unmarshal(b, &user); err != nil {
			return nil, fmt.Errorf("recipe: %s is not valid JSON: %w", path, err)
		}
		for _, r := range user.Recipes {
			merged[r.Name] = r // user overrides seed by name
		}
	}
	return merged, nil
}

// RunRecipe executes the named recipe. vars supplies $VAR values;
// confirm allows Confirm-gated (submit) steps to fire. It never aborts
// the whole run on an Optional step's failure; a non-optional failure
// stops and returns the partial result plus the error.
func RunRecipe(name string, vars map[string]string, confirm bool) (RecipeResult, error) {
	return runRecipeWith(defaultAdapter, name, vars, confirm)
}

// defaultRecipeStepDelayMS is a settle between recipe steps. Live
// testing showed that rapid back-to-back UI actions race the target's
// first-responder/selection state — a click, then an immediate cmd+a,
// then a type, drops the keystrokes (the window is focused but the text
// view hasn't settled). A brief per-step pause makes replay reliable,
// mirroring the hand-paced sleeps a human would have. Override with
// WCTL_RECIPE_STEP_DELAY_MS (0 disables). Read per run so it's tunable
// without a rebuild.
const defaultRecipeStepDelayMS = 250

func recipeStepDelay() time.Duration {
	if v, ok := os.LookupEnv("WCTL_RECIPE_STEP_DELAY_MS"); ok {
		if ms, err := strconv.Atoi(v); err == nil && ms >= 0 {
			return msDuration(ms)
		}
	}
	return msDuration(defaultRecipeStepDelayMS)
}

func runRecipeWith(a Adapter, name string, vars map[string]string, confirm bool) (RecipeResult, error) {
	r, err := GetRecipe(name)
	if err != nil {
		return RecipeResult{}, err
	}
	res := RecipeResult{Name: name}
	delay := recipeStepDelay()
	for i, step := range r.Steps {
		if i > 0 && delay > 0 {
			sleepFn(delay)
		}
		if step.Confirm && !confirm {
			res.HeldForConfirm = true
			res.Steps = append(res.Steps, StepOutcome{Action: step.Action, Status: "skipped", Detail: "confirm-gated (submit) — pass --confirm to run"})
			continue
		}
		detail, err := runStep(a, r, step, vars)
		if err != nil {
			if step.Optional {
				res.Steps = append(res.Steps, StepOutcome{Action: step.Action, Status: "skipped", Detail: "optional: " + err.Error()})
				continue
			}
			res.Steps = append(res.Steps, StepOutcome{Action: step.Action, Status: "failed", Detail: err.Error()})
			return res, fmt.Errorf("recipe %q: step %q failed: %w", name, step.Action, err)
		}
		res.Steps = append(res.Steps, StepOutcome{Action: step.Action, Status: "ok", Detail: detail})
	}
	return res, nil
}

// stepMatch resolves the window scope for a step: its own App/Title
// override the recipe's Match.
func stepMatch(r Recipe, s RecipeStep) Match {
	m := Match{Title: r.Match.Title, App: r.Match.App}
	if s.App != "" {
		m.App = s.App
	}
	if s.Title != "" {
		m.Title = s.Title
	}
	return m
}

func runStep(a Adapter, r Recipe, s RecipeStep, vars map[string]string) (string, error) {
	m := stepMatch(r, s)
	text, err := substituteVars(s.Text, r.Vars, vars)
	if err != nil {
		return "", err
	}
	button := parseButton(s.Button)

	switch s.Action {
	case "focus":
		return "", focusWith(a, m)
	case "launch":
		app := s.App
		if app == "" {
			app = m.App
		}
		if app == "" {
			return "", errors.New("launch: no app to launch")
		}
		return app, a.Launch(app)
	case "wait-text":
		to := s.TimeoutMS
		if to <= 0 {
			to = 6000
		}
		return waitTextWith(a, FindOptions{Text: text, Match: matchScope(m)}, to)
	case "find-click":
		return findClickWith(a, m, text, s.X, s.Y)
	case "click":
		if s.X == nil || s.Y == nil {
			return "", errors.New("click: x and y are required")
		}
		return fmt.Sprintf("%d,%d", *s.X, *s.Y), mouseClickWith(a, ClickOptions{X: s.X, Y: s.Y, Button: button})
	case "type":
		return typeStep(a, m, text, s.Method)
	case "paste":
		return "paste", pasteInto(a, m, text)
	case "key":
		if s.Combo == "" {
			return "", errors.New("key: combo is required")
		}
		if m.Title == "" && m.App == "" {
			return s.Combo, PressKey(s.Combo)
		}
		return s.Combo, pressKeyIntoWith(a, m, s.Combo)
	case "scroll":
		return fmt.Sprintf("dx=%d dy=%d", s.DX, s.DY), scrollWith(a, ScrollOptions{X: s.X, Y: s.Y, DX: s.DX, DY: s.DY})
	case "drag":
		if s.X == nil || s.Y == nil || s.ToX == nil || s.ToY == nil {
			return "", errors.New("drag: x, y, to_x, to_y are required")
		}
		return "", dragWith(a, DragOptions{FromX: *s.X, FromY: *s.Y, ToX: *s.ToX, ToY: *s.ToY, Button: button})
	default:
		return "", fmt.Errorf("unknown action %q", s.Action)
	}
}

// matchScope returns a *Match for FindOptions when the scope is set,
// else nil (whole focused monitor).
func matchScope(m Match) *Match {
	if m.Title == "" && m.App == "" {
		return nil
	}
	return &m
}

// findClickWith is the vision-first click: OCR the scope for text and
// click the top match; if OCR finds nothing, fall back to the (X,Y)
// coordinate if the step provided one. "Allow all ways."
func findClickWith(a Adapter, m Match, text string, fx, fy *int) (string, error) {
	matches, err := findTextWith(a, FindOptions{Text: text, Match: matchScope(m)})
	if err != nil {
		return "", err
	}
	if len(matches) > 0 {
		top := matches[0]
		if err := mouseClickWith(a, ClickOptions{X: &top.ClickX, Y: &top.ClickY}); err != nil {
			return "", err
		}
		return fmt.Sprintf("vision %q@%d,%d", text, top.ClickX, top.ClickY), nil
	}
	if fx != nil && fy != nil {
		if err := mouseClickWith(a, ClickOptions{X: fx, Y: fy}); err != nil {
			return "", err
		}
		return fmt.Sprintf("coord-fallback %d,%d (OCR found no %q)", *fx, *fy, text), nil
	}
	return "", fmt.Errorf("find-click: no on-screen text matched %q and no fallback x/y given", text)
}

// typeStep types text with the requested method. keystroke (default/
// auto) uses the focus-verified type path; paste uses the clipboard.
// "auto" tries keystroke and, iff it yields an error, falls back to
// paste — layout/keycode-proof.
func typeStep(a Adapter, m Match, text, method string) (string, error) {
	switch method {
	case "paste":
		return "paste", pasteInto(a, m, text)
	case "", "keystroke", "auto":
		err := typeIntoWith(a, m, text)
		if err != nil && method == "auto" {
			if perr := pasteInto(a, m, text); perr == nil {
				return "keystroke→paste fallback", nil
			}
		}
		return "keystroke", err
	default:
		return "", fmt.Errorf("type: unknown method %q (want keystroke|paste|auto)", method)
	}
}

// pasteInto sets the clipboard then presses cmd+v into the matched
// window (focus-verified). The reliable, layout-independent write.
func pasteInto(a Adapter, m Match, text string) error {
	if err := a.SetClipboard(text); err != nil {
		return err
	}
	if m.Title == "" && m.App == "" {
		return PressKey("cmd+v")
	}
	return pressKeyIntoWith(a, m, "cmd+v")
}

// waitTextWith polls find --text until the query appears or the timeout
// elapses. Uses the same test-overridable clock as WaitForWindow.
func waitTextWith(a Adapter, opts FindOptions, timeoutMS int) (string, error) {
	deadline := nowFn().Add(msDuration(timeoutMS))
	for {
		matches, err := findTextWith(a, opts)
		if err != nil {
			return "", err
		}
		if len(matches) > 0 {
			return fmt.Sprintf("%q appeared", opts.Text), nil
		}
		if !nowFn().Before(deadline) {
			return "", fmt.Errorf("wait-text: %q did not appear within %dms", opts.Text, timeoutMS)
		}
		sleepFn(waitPollInterval)
	}
}

func parseButton(s string) MouseButton {
	switch s {
	case "right":
		return MouseRight
	case "middle":
		return MouseMiddle
	default:
		return MouseLeft
	}
}

// substituteVars replaces $NAME and ${NAME} in s with values from vars.
// Only names declared in declared are eligible; a declared var with no
// provided value is an error (so a recipe never silently types "$TEXT").
func substituteVars(s string, declared []string, vars map[string]string) (string, error) {
	if s == "" {
		return "", nil
	}
	out := s
	for _, name := range declared {
		val, ok := vars[name]
		if !ok {
			// Only error if the placeholder is actually used.
			if strings.Contains(out, "$"+name) || strings.Contains(out, "${"+name+"}") {
				return "", fmt.Errorf("recipe: missing --var %s", name)
			}
			continue
		}
		out = strings.ReplaceAll(out, "${"+name+"}", val)
		out = strings.ReplaceAll(out, "$"+name, val)
	}
	return out, nil
}
