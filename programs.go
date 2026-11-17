package main

import (
	"embed"
	"path"
	"strings"

	"github.com/egoist/mygo/ui"
)

//go:embed assets/icons/*.svg assets/brands/*.svg assets/logos/*.svg
var assets embed.FS

var svgs = map[string]*ui.SVG{}

// icon returns an embedded SVG: "x" from assets/icons, "brand:git" from
// assets/brands, or "logo:near-ai-secondary-color" from assets/logos.
func icon(name string) *ui.SVG {
	if s, ok := svgs[name]; ok {
		return s
	}
	file := "assets/icons/" + name + ".svg"
	if b, ok := strings.CutPrefix(name, "brand:"); ok {
		file = "assets/brands/" + b + ".svg"
	} else if l, ok := strings.CutPrefix(name, "logo:"); ok {
		file = "assets/logos/" + l + ".svg"
	}
	data, err := assets.ReadFile(file)
	if err != nil {
		panic(err)
	}
	s := ui.MustParseSVG(data)
	svgs[name] = s
	return s
}

// nearSecondaryLogo returns the official NEAR AI secondary lockup matched to
// the active theme's light/dark appearance (unchanged SVG artwork per NDS-8).
func nearSecondaryLogo(dark bool) *ui.SVG {
	if dark {
		return icon("logo:near-ai-secondary-white-color")
	}
	return icon("logo:near-ai-secondary-color")
}

// program describes how a foreground CLI/shell process renders in headers and
// flat Near AI OS tool tiles (08-iconography.md: flat neutral-tint square,
// radius-tile = 6, >= 3:1 glyph tint in light and dark, no top sheen).
type program struct {
	Name      string
	Glyph     string
	TileGlyph string
	TintLight ui.Color
	TintDark  ui.Color
	DataIndex int // >= 0 selects Tokens.data[DataIndex] when TintLight.A == 0
	Shell     bool
}

func (p program) glyphTint(ts *themeState) ui.Color {
	if p.TintLight.A == 0 {
		return ts.data[p.DataIndex%len(ts.data)]
	}
	if ts.Dark {
		return p.TintDark
	}
	return p.TintLight
}

var (
	shellTile = program{
		Glyph:     "square-terminal",
		TileGlyph: "terminal",
		TintLight: ui.Hex("#1a8f4c"),
		TintDark:  ui.Hex("#30d158"),
		Shell:     true,
	}

	programs = map[string]program{
		"codex":     {Name: "Codex", Glyph: "brand:openai", TintLight: ui.Hex("#1d1d1f"), TintDark: ui.Hex("#f2f3f8")},
		"claude":    {Name: "Claude Code", Glyph: "brand:claude", TintLight: ui.Hex("#b8532f"), TintDark: ui.Hex("#f08c69")},
		"node":      {Name: "Node", Glyph: "node-hex", TileGlyph: "brand:nodedotjs", TintLight: ui.Hex("#267339"), TintDark: ui.Hex("#5fd38d")},
		"bun":       {Name: "Bun", Glyph: "brand:bun", TintLight: ui.Hex("#7a5019"), TintDark: ui.Hex("#f5c98c")},
		"deno":      {Name: "Deno", Glyph: "brand:deno", TintLight: ui.Hex("#1d1d1f"), TintDark: ui.Hex("#f2f3f8")},
		"python":    {Name: "Python", Glyph: "brand:python", TintLight: ui.Hex("#1f5f99"), TintDark: ui.Hex("#7ebcff")},
		"lazygit":   {Name: "Git Changes", Glyph: "plus-minus-circle", TileGlyph: "diff", TintLight: ui.Hex("#1a8f4c"), TintDark: ui.Hex("#30d158")},
		"tig":       {Name: "Git Log", Glyph: "git-branch", TintLight: ui.Hex("#c94124"), TintDark: ui.Hex("#ff7b61")},
		"git":       {Name: "Git", Glyph: "brand:git", TintLight: ui.Hex("#c94124"), TintDark: ui.Hex("#ff7b61")},
		"vim":       {Name: "Vim", Glyph: "brand:vim", TintLight: ui.Hex("#187d38"), TintDark: ui.Hex("#4ade80")},
		"nvim":      {Name: "Neovim", Glyph: "brand:neovim", TintLight: ui.Hex("#1c7542"), TintDark: ui.Hex("#5eead4")},
		"hx":        {Name: "Helix", Glyph: "brand:helix", TintLight: ui.Hex("#5a48b8"), TintDark: ui.Hex("#c4b5fd")},
		"emacs":     {Name: "Emacs", Glyph: "brand:gnuemacs", TintLight: ui.Hex("#5a48b8"), TintDark: ui.Hex("#b794f4")},
		"htop":      {Name: "Activity", Glyph: "activity", TintLight: ui.Hex("#147d5b"), TintDark: ui.Hex("#38d9a9")},
		"btop":      {Name: "Activity", Glyph: "activity", TintLight: ui.Hex("#147d5b"), TintDark: ui.Hex("#38d9a9")},
		"top":       {Name: "Activity", Glyph: "activity", TintLight: ui.Hex("#147d5b"), TintDark: ui.Hex("#38d9a9")},
		"ssh":       {Name: "SSH", Glyph: "globe", TintLight: ui.Hex("#035ca3"), TintDark: ui.Hex("#7ebcff")},
		"mosh":      {Name: "Mosh", Glyph: "globe", TintLight: ui.Hex("#035ca3"), TintDark: ui.Hex("#7ebcff")},
		"docker":    {Name: "Docker", Glyph: "brand:docker", TintLight: ui.Hex("#0264b5"), TintDark: ui.Hex("#5cb8ff")},
		"go":        {Name: "Go", Glyph: "brand:go", TintLight: ui.Hex("#006d8f"), TintDark: ui.Hex("#5ce1e6")},
		"cargo":     {Name: "Cargo", Glyph: "brand:rust", TintLight: ui.Hex("#9e4716"), TintDark: ui.Hex("#f4a261")},
		"rustc":     {Name: "Rust", Glyph: "brand:rust", TintLight: ui.Hex("#9e4716"), TintDark: ui.Hex("#f4a261")},
		"npm":       {Name: "npm", Glyph: "brand:npm", TintLight: ui.Hex("#b82928"), TintDark: ui.Hex("#ff6b6b")},
		"pnpm":      {Name: "pnpm", Glyph: "brand:pnpm", TintLight: ui.Hex("#a85d00"), TintDark: ui.Hex("#ffc75c")},
		"yarn":      {Name: "Yarn", Glyph: "brand:yarn", TintLight: ui.Hex("#196b91"), TintDark: ui.Hex("#74c0fc")},
		"ruby":      {Name: "Ruby", Glyph: "brand:ruby", TintLight: ui.Hex("#b52822"), TintDark: ui.Hex("#ff7575")},
		"irb":       {Name: "Ruby", Glyph: "brand:ruby", TintLight: ui.Hex("#b52822"), TintDark: ui.Hex("#ff7575")},
		"lua":       {Name: "Lua", Glyph: "brand:lua", TintLight: ui.Hex("#2c3e99"), TintDark: ui.Hex("#91a7ff")},
		"php":       {Name: "PHP", Glyph: "brand:php", TintLight: ui.Hex("#4c5194"), TintDark: ui.Hex("#a5b4fc")},
		"psql":      {Name: "PostgreSQL", Glyph: "brand:postgresql", TintLight: ui.Hex("#235784"), TintDark: ui.Hex("#7ebcff")},
		"mysql":     {Name: "MySQL", Glyph: "brand:mysql", TintLight: ui.Hex("#00647d"), TintDark: ui.Hex("#66d9e8")},
		"redis-cli": {Name: "Redis", Glyph: "brand:redis", TintLight: ui.Hex("#bd2419"), TintDark: ui.Hex("#ff6b6b")},
		"tmux":      {Name: "tmux", Glyph: "brand:tmux", TintLight: ui.Hex("#187d38"), TintDark: ui.Hex("#30d158")},
		"make":      {Name: "Make", Glyph: "cpu", TintLight: ui.Hex("#4b5563"), TintDark: ui.Hex("#cbd5e1")},
		"swift":     {Name: "Swift", Glyph: "brand:swift", TintLight: ui.Hex("#c73d26"), TintDark: ui.Hex("#ff7f66")},
		"kotlin":    {Name: "Kotlin", Glyph: "brand:kotlin", TintLight: ui.Hex("#6338d9"), TintDark: ui.Hex("#b197fc")},
	}

	shells = map[string]bool{
		"zsh": true, "bash": true, "fish": true, "sh": true, "dash": true,
		"nu": true, "pwsh": true, "elvish": true, "xonsh": true, "tcsh": true,
		"csh": true, "ksh": true, "cmd": true, "powershell": true,
	}
)

func init() {
	programs["python3"] = programs["python"]
	programs["ipython"] = programs["python"]
	programs["view"] = programs["vim"]
	programs["vi"] = programs["vim"]
	programs["less"] = program{Name: "Pager", Glyph: "file-code", TintLight: ui.Hex("#3c4b5e"), TintDark: ui.Hex("#cbd5e1")}
	programs["man"] = programs["less"]
}

// programOf returns how a program renders by its process name.
func programOf(name string) program {
	name = strings.TrimPrefix(path.Base(name), "-")
	if p, ok := programs[strings.ToLower(name)]; ok {
		return p
	}
	if shells[name] {
		p := shellTile
		p.Name = name
		return p
	}
	return program{
		Name:      name,
		Glyph:     "square-terminal",
		TileGlyph: "terminal",
		DataIndex: hashIndex(name, 8),
	}
}

func hashIndex(s string, n int) int {
	h := uint32(2166136261)
	for i := 0; i < len(s); i++ {
		h ^= uint32(s[i])
		h *= 16777619
	}
	return int(h % uint32(n))
}
