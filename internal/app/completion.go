// Copyright (c) 2026 Adam Snyder <https://armsnyder.com> and contributors
// SPDX-License-Identifier: MIT

package app

import (
	"embed"
	"encoding/csv"
	"fmt"
	"io/fs"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/armsnyder/gdshader-language-server/internal/lsp"
)

//go:embed tables
var tablesFS embed.FS

type completionContext struct {
	shaderType   string
	functionName string
	lineTokens   []string
}

func (c completionContext) lastToken() string {
	if len(c.lineTokens) == 0 {
		return ""
	}
	return c.lineTokens[len(c.lineTokens)-1]
}

type completionPredicate func(c completionContext) bool

type completionItemPredicate struct {
	predicate completionPredicate
	item      lsp.CompletionItem
}

type tableData struct {
	header []string
	rows   [][]string
}

var boldNamePattern = regexp.MustCompile(`\*\*([A-Za-z_][A-Za-z0-9_]*)\*\*`)

var completionItems = buildCompletionItems()

// buildCompletionItems composes all table-derived and static completions once at startup.
func buildCompletionItems() []completionItemPredicate {
	var items []completionItemPredicate

	items = append(items, staticKeywordItems()...)
	items = append(items, shaderTypeItems()...)
	items = append(items, dataTypeItems()...)
	items = append(items, interpolationQualifierItems()...)
	items = append(items, uniformHintItems()...)

	shaders, err := shaderKinds()
	if err != nil {
		panic(err)
	}
	for _, shader := range shaders {
		items = append(items, shaderTableItems(shader)...)
	}

	return dedupeCompletionItems(items)
}

func staticKeywordItems() []completionItemPredicate {
	return []completionItemPredicate{
		makeKeywordItem("shader_type", ifIsFirst, "Declares the shader type, such as `canvas_item`, `spatial`, or `particles`."),
		makeKeywordItem("render_mode", ifIsFirst, "Declares one or more render modes for the shader."),
		makeKeywordItem("uniform", ifIsFirst, "Declares a variable set from outside the shader."),
		makeKeywordItem("varying", ifIsFirst, "Declares a variable passed between shader stages."),
		makeKeywordItem("group_uniforms", ifIsFirst, "Groups uniforms together in the inspector."),
		makeKeywordItem("in", ifLastTokenOneOf("(", ","), "Function argument qualifier: read-only parameter."),
		makeKeywordItem("out", ifLastTokenOneOf("(", ","), "Function argument qualifier: write-only parameter."),
		makeKeywordItem("inout", ifLastTokenOneOf("(", ","), "Function argument qualifier: read-write parameter."),
		makeKeywordItem("flat", ifLastTokenOneOf("varying"), "Interpolation qualifier: no interpolation."),
		makeKeywordItem("smooth", ifLastTokenOneOf("varying"), "Interpolation qualifier: perspective-correct interpolation."),
		makeKeywordItem("lowp", alwaysTrue, "low precision, usually 8 bits per component mapped to 0-1"),
		makeKeywordItem("mediump", alwaysTrue, "medium precision, usually 16 bits or half float"),
		makeKeywordItem("highp", alwaysTrue, "high precision, uses full float or integer range (32 bit default)"),
		makeKeywordItem("const", alwaysTrue, ""),
		makeKeywordItem("struct", alwaysTrue, ""),
		makeKeywordItem("break", alwaysTrue, ""),
		makeKeywordItem("case", alwaysTrue, ""),
		makeKeywordItem("continue", alwaysTrue, ""),
		makeKeywordItem("default", alwaysTrue, ""),
		makeKeywordItem("do", alwaysTrue, ""),
		makeKeywordItem("else", alwaysTrue, ""),
		makeKeywordItem("for", alwaysTrue, ""),
		makeKeywordItem("if", alwaysTrue, ""),
		makeKeywordItem("return", alwaysTrue, ""),
		makeKeywordItem("switch", alwaysTrue, ""),
		makeKeywordItem("while", alwaysTrue, ""),
	}
}

// shaderTypeItems discovers available shader kinds from embedded table directories.
func shaderTypeItems() []completionItemPredicate {
	shaders, err := shaderKinds()
	if err != nil {
		panic(err)
	}

	items := make([]completionItemPredicate, 0, len(shaders))
	for _, shader := range shaders {
		doc := shaderTypeDocumentation(shader)
		var documentation *lsp.MarkupContent
		if doc != "" {
			documentation = &lsp.MarkupContent{
				Kind:  lsp.MarkupMarkdown,
				Value: doc,
			}
		}
		items = append(items, completionItemPredicate{
			predicate: ifLastTokenOneOf("shader_type"),
			item: lsp.CompletionItem{
				Label:         shader,
				Kind:          lsp.CompletionKeyword,
				Documentation: documentation,
			},
		})
	}
	return items
}

// shaderTypeDocumentation returns hover text for known shader types.
func shaderTypeDocumentation(shader string) string {
	switch shader {
	case "canvas_item":
		return "Canvas item shader, used for 2D rendering."
	case "spatial":
		return "Spatial shader, used for 3D rendering."
	case "particles":
		return "Particle shader, used for particle systems."
	case "sky":
		return "Sky shader, used for rendering skyboxes or skydomes."
	case "fog":
		return "Fog shader, used for rendering fog effects."
	default:
		return ""
	}
}

// dataTypeItems maps shading-language data type table rows to completion docs.
func dataTypeItems() []completionItemPredicate {
	table, err := readTableData("tables/shading_language/data_types.csv")
	if err != nil {
		panic(err)
	}
	gdscriptTypes, err := gdscriptTypeByGLSLType()
	if err != nil {
		panic(err)
	}

	items := make([]completionItemPredicate, 0, len(table.rows))
	for _, row := range table.rows {
		if len(row) < 2 {
			continue
		}
		label := extractBoldName(row[0])
		if label == "" {
			continue
		}
		items = append(items, completionItemPredicate{
			predicate: or(
				ifLastTokenOneOf("uniform", "varying", "in", "out", "inout", "flat", "smooth", "lowp", "mediump", "highp"),
				isLastTokenPunctuation,
			),
			item: lsp.CompletionItem{
				Label: label,
				Kind:  lsp.CompletionClass,
				Documentation: &lsp.MarkupContent{
					Kind:  lsp.MarkupMarkdown,
					Value: dataTypeDocumentation(label, cleanDocText(row[1]), gdscriptTypes),
				},
			},
		})
	}

	return items
}

// gdscriptTypeByGLSLType maps GLSL types to the corresponding GDScript type descriptions.
func gdscriptTypeByGLSLType() (map[string]string, error) {
	table, err := readTableData("tables/shading_language/setting_uniforms_from_code.csv")
	if err != nil {
		return nil, err
	}

	types := make(map[string]string, len(table.rows))
	for _, row := range table.rows {
		if len(row) < 2 {
			continue
		}
		glslType := extractBoldName(row[0])
		gdscriptType := cleanDocText(row[1])
		if glslType == "" || gdscriptType == "" {
			continue
		}
		types[glslType] = gdscriptType
	}
	return types, nil
}

// dataTypeDocumentation returns data-type hover docs, optionally including GDScript mapping info.
func dataTypeDocumentation(label, baseDoc string, gdscriptTypes map[string]string) string {
	gdscriptType := gdscriptTypes[label]
	if gdscriptType == "" {
		return baseDoc
	}
	return baseDoc + "\n\nGDScript type: " + gdscriptType
}

// interpolationQualifierItems maps qualifier rows to varying-context completions.
func interpolationQualifierItems() []completionItemPredicate {
	table, err := readTableData("tables/shading_language/interpolation_qualifiers.csv")
	if err != nil {
		panic(err)
	}

	items := make([]completionItemPredicate, 0, len(table.rows))
	for _, row := range table.rows {
		if len(row) < 2 {
			continue
		}
		label := extractBoldName(row[0])
		if label == "" {
			continue
		}
		items = append(items, completionItemPredicate{
			predicate: ifLastTokenOneOf("varying"),
			item: lsp.CompletionItem{
				Label: label,
				Kind:  lsp.CompletionKeyword,
				Documentation: &lsp.MarkupContent{
					Kind:  lsp.MarkupMarkdown,
					Value: cleanDocText(row[1]),
				},
			},
		})
	}

	return items
}

// uniformHintItems turns uniform hint table rows into hint completions after ':' in uniform declarations.
func uniformHintItems() []completionItemPredicate {
	table, err := readTableData("tables/shading_language/uniform_hints.csv")
	if err != nil {
		panic(err)
	}

	items := make([]completionItemPredicate, 0, len(table.rows))
	for _, row := range table.rows {
		if len(row) < 3 {
			continue
		}
		validTypes := uniformHintTypes(row[0])
		if len(validTypes) == 0 {
			continue
		}
		hint := cleanDocText(row[1])
		if hint == "" {
			continue
		}
		for _, expandedHint := range expandBracketHintPattern(hint) {
			label := expandedHint
			kind := lsp.CompletionKeyword
			if i := strings.Index(expandedHint, "("); i > 0 {
				// Function-like hints are completed by name, while full signature remains in detail.
				label = expandedHint[:i]
				kind = lsp.CompletionFunction
			}
			items = append(items, completionItemPredicate{
				predicate: and(
					ifFirstTokenOneOf("uniform"),
					ifTokensContain(":"),
					ifUniformTypeOneOf(validTypes...),
				),
				item: lsp.CompletionItem{
					Label:  label,
					Kind:   kind,
					Detail: expandedHint,
					Documentation: &lsp.MarkupContent{
						Kind:  lsp.MarkupMarkdown,
						Value: cleanDocText(row[2]),
					},
				},
			})
		}
	}
	return items
}

// expandBracketHintPattern expands Godot hint patterns like repeat[_enable, _disable].
func expandBracketHintPattern(pattern string) []string {
	variants := []string{""}
	remaining := pattern

	for {
		group := parseNextBracketGroup(remaining)
		appendSuffix(variants, group.prefix)
		if !group.found {
			appendSuffix(variants, group.rest)
			return variants
		}

		if len(group.options) == 1 {
			// Single-option groups are optional in docs notation, e.g. [_mipmap].
			group.options = append(group.options, "")
		}
		variants = combineVariants(variants, group.options)
		remaining = group.rest
	}
}

type bracketGroup struct {
	prefix  string
	options []string
	rest    string
	found   bool
}

// parseNextBracketGroup splits the next [a, b, ...] group and returns text before it.
func parseNextBracketGroup(s string) bracketGroup {
	open := strings.IndexByte(s, '[')
	if open < 0 {
		return bracketGroup{prefix: s, found: false}
	}
	closeIdx := strings.IndexByte(s[open+1:], ']')
	if closeIdx < 0 {
		return bracketGroup{prefix: s, found: false}
	}
	closeIdx += open + 1
	return bracketGroup{
		prefix:  s[:open],
		options: splitPatternOptions(s[open+1 : closeIdx]),
		rest:    s[closeIdx+1:],
		found:   true,
	}
}

func appendSuffix(variants []string, suffix string) {
	for i := range variants {
		variants[i] += suffix
	}
}

func combineVariants(variants, options []string) []string {
	next := make([]string, 0, len(variants)*len(options))
	for _, variant := range variants {
		for _, option := range options {
			next = append(next, variant+option)
		}
	}
	return next
}

func splitPatternOptions(s string) []string {
	raw := strings.Split(s, ",")
	out := make([]string, 0, len(raw))
	for _, option := range raw {
		trimmed := strings.TrimSpace(option)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

// shaderTableItems loads per-shader table files and dispatches each file class to a builder.
func shaderTableItems(shader string) []completionItemPredicate {
	dir := shaderToTableDir(shader)
	entries, err := fs.ReadDir(tablesFS, filepath.Join("tables", dir))
	if err != nil {
		panic(err)
	}

	var items []completionItemPredicate
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		switch {
		case strings.HasSuffix(name, "_modes.csv"):
			modeKeyword := modeKeywordForFileName(name)
			items = append(items, modeItems(shader, modeKeyword, filepath.Join("tables", dir, name))...)
		case strings.HasSuffix(name, "_builtins.csv"):
			items = append(items, builtinItems(shader, name, filepath.Join("tables", dir, name))...)
		case name == "normal.csv":
			items = append(items, builtinItems(shader, "normal_builtins.csv", filepath.Join("tables", dir, name))...)
		case strings.HasSuffix(name, "_functions.csv"):
			items = append(items, functionItems(shader, filepath.Join("tables", dir, name))...)
		}
	}

	return items
}

// modeItems maps mode rows into shader-scoped mode completions for modeKeyword.
func modeItems(shader, modeKeyword, path string) []completionItemPredicate {
	table, err := readTableData(path)
	if err != nil {
		panic(err)
	}

	items := make([]completionItemPredicate, 0, len(table.rows))
	items = append(items, makeKeywordItem(
		modeKeyword,
		and(ifShaderType(shader), ifIsFirst),
		modeKeywordDocumentation(modeKeyword),
	))
	for _, row := range table.rows {
		if len(row) < 2 {
			continue
		}
		label := extractBoldName(row[0])
		if label == "" {
			label = cleanDocText(row[0])
		}
		if label == "" {
			continue
		}
		items = append(items, completionItemPredicate{
			predicate: and(ifShaderType(shader), ifFirstTokenOneOf(modeKeyword)),
			item: lsp.CompletionItem{
				Label: label,
				Kind:  lsp.CompletionKeyword,
				Documentation: &lsp.MarkupContent{
					Kind:  lsp.MarkupMarkdown,
					Value: cleanDocText(row[1]),
				},
			},
		})
	}
	return items
}

func modeKeywordForFileName(name string) string {
	switch name {
	case "stencil_modes.csv":
		return "stencil_mode"
	default:
		return "render_mode"
	}
}

func modeKeywordDocumentation(modeKeyword string) string {
	switch modeKeyword {
	case "stencil_mode":
		return "Declares one or more stencil modes for the shader."
	default:
		return "Declares one or more render modes for the shader."
	}
}

// builtinItems maps built-in symbol rows into shader/stage-scoped constant completions.
func builtinItems(shader, fileName, path string) []completionItemPredicate {
	table, err := readTableData(path)
	if err != nil {
		panic(err)
	}

	predicate := ifShaderType(shader)
	switch stageNameFromBuiltinsFile(fileName) {
	case "":
		// global built-ins: no extra stage predicate.
	default:
		// Filenames like "start_and_process_builtins.csv" encode multiple function scopes.
		stages := strings.Split(stageNameFromBuiltinsFile(fileName), "_and_")
		predicate = and(predicate, inAnyFunction(stages...))
	}

	items := make([]completionItemPredicate, 0, len(table.rows))
	for _, row := range table.rows {
		if len(row) < 2 {
			continue
		}
		label := extractBoldName(row[0])
		if label == "" {
			continue
		}
		items = append(items, completionItemPredicate{
			predicate: predicate,
			item: lsp.CompletionItem{
				Label:  label,
				Kind:   lsp.CompletionConstant,
				Detail: cleanDocText(row[0]),
				Documentation: &lsp.MarkupContent{
					Kind:  lsp.MarkupMarkdown,
					Value: cleanDocText(row[1]),
				},
			},
		})
	}
	return items
}

// functionItems maps function signature rows into shader-scoped function completions.
func functionItems(shader, path string) []completionItemPredicate {
	table, err := readTableData(path)
	if err != nil {
		panic(err)
	}

	items := make([]completionItemPredicate, 0, len(table.rows))
	for _, row := range table.rows {
		if len(row) < 2 {
			continue
		}
		label := extractBoldName(row[0])
		if label == "" {
			continue
		}
		signature := cleanDocText(row[0])
		items = append(items, completionItemPredicate{
			predicate: ifShaderType(shader),
			item: lsp.CompletionItem{
				Label:  label,
				Kind:   lsp.CompletionFunction,
				Detail: signature,
				Documentation: &lsp.MarkupContent{
					Kind:  lsp.MarkupMarkdown,
					Value: cleanDocText(row[1]),
				},
			},
		})
	}
	return items
}

// dedupeCompletionItems removes duplicate labels while preserving first-seen ordering.
func dedupeCompletionItems(items []completionItemPredicate) []completionItemPredicate {
	seen := make(map[string]int, len(items))
	out := make([]completionItemPredicate, 0, len(items))
	for _, item := range items {
		// Include detail and documentation so context-specific docs don't collapse together.
		key := item.item.Label + "\x00" + item.item.Detail + "\x00" + completionDocValue(item.item.Documentation)
		if idx, ok := seen[key]; ok {
			// Same label/detail across different shader/stage scopes should accumulate predicates.
			out[idx].predicate = or(out[idx].predicate, item.predicate)
			if out[idx].item.Documentation == nil && item.item.Documentation != nil {
				out[idx].item.Documentation = item.item.Documentation
			}
			continue
		}
		seen[key] = len(out)
		out = append(out, item)
	}
	return out
}

func completionDocValue(doc *lsp.MarkupContent) string {
	if doc == nil {
		return ""
	}
	return doc.Value
}

// shaderKinds returns shader names based on embedded *_shader table directories.
func shaderKinds() ([]string, error) {
	entries, err := fs.ReadDir(tablesFS, "tables")
	if err != nil {
		return nil, fmt.Errorf("read tables dir: %w", err)
	}

	var shaders []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, "_shader") {
			continue
		}
		shader := strings.TrimSuffix(name, "_shader")
		// Match shader_type keyword spelling used by Godot.
		if shader == "particle" {
			shader = "particles"
		}
		shaders = append(shaders, shader)
	}

	slices.Sort(shaders)
	return shaders, nil
}

// shaderToTableDir converts a shader_type value into its table directory name.
func shaderToTableDir(shader string) string {
	if shader == "particles" {
		return "particle_shader"
	}
	return shader + "_shader"
}

// stageNameFromBuiltinsFile extracts the optional function scope encoded in *_builtins.csv filenames.
func stageNameFromBuiltinsFile(fileName string) string {
	switch fileName {
	case "global_builtins.csv":
		return ""
	default:
		return strings.TrimSuffix(fileName, "_builtins.csv")
	}
}

// readTableData reads one embedded CSV table and splits header and body records.
func readTableData(path string) (tableData, error) {
	f, err := tablesFS.Open(path)
	if err != nil {
		return tableData{}, err
	}
	defer func() { _ = f.Close() }()

	r := csv.NewReader(f)
	r.FieldsPerRecord = -1
	records, err := r.ReadAll()
	if err != nil {
		return tableData{}, err
	}
	if len(records) == 0 {
		return tableData{}, fmt.Errorf("empty table: %s", path)
	}
	return tableData{
		header: records[0],
		rows:   records[1:],
	}, nil
}

// extractBoldName returns the first symbol wrapped in **...** used by Godot docs tables.
func extractBoldName(s string) string {
	match := boldNamePattern.FindStringSubmatch(s)
	if len(match) != 2 {
		return ""
	}
	return match[1]
}

// uniformHintTypes parses the "Type" column from uniform_hints.csv into concrete type tokens.
func uniformHintTypes(s string) []string {
	s = strings.ReplaceAll(s, "**", "")
	parts := strings.Split(s, ",")
	types := make([]string, 0, len(parts))
	for _, part := range parts {
		t := strings.TrimSpace(part)
		if t != "" {
			types = append(types, t)
		}
	}
	return types
}

// cleanDocText normalizes lightweight RST-like markers for hover/completion display.
func cleanDocText(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "**", "")
	s = strings.ReplaceAll(s, "``", "`")
	return s
}

func alwaysTrue(completionContext) bool { return true }

func ifLastTokenOneOf(tokens ...string) completionPredicate {
	return func(c completionContext) bool {
		return slices.Contains(tokens, c.lastToken())
	}
}

func ifFirstTokenOneOf(tokens ...string) completionPredicate {
	return func(c completionContext) bool {
		if len(c.lineTokens) == 0 {
			return false
		}
		return slices.Contains(tokens, c.lineTokens[0])
	}
}

func ifIsFirst(c completionContext) bool {
	return len(c.lineTokens) == 0
}

func ifShaderType(shaderType string) completionPredicate {
	return func(c completionContext) bool {
		return c.shaderType == shaderType
	}
}

// inAnyFunction matches listed stage entrypoints and also matches helper functions.
func inAnyFunction(names ...string) completionPredicate {
	return func(c completionContext) bool {
		if c.functionName == "" {
			return false
		}
		if slices.Contains(names, c.functionName) {
			return true
		}
		return !isEntrypointFunction(c.functionName)
	}
}

// isEntrypointFunction reports whether name is a Godot stage entrypoint function.
func isEntrypointFunction(name string) bool {
	switch name {
	case "vertex", "fragment", "light", "start", "process", "fog", "sky":
		return true
	default:
		return false
	}
}

func ifTokensContain(token string) completionPredicate {
	return func(c completionContext) bool {
		return slices.Contains(c.lineTokens, token)
	}
}

// ifUniformTypeOneOf matches when a uniform declaration contains one of the provided type tokens.
func ifUniformTypeOneOf(types ...string) completionPredicate {
	return func(c completionContext) bool {
		colonIndex := slices.Index(c.lineTokens, ":")
		if colonIndex <= 1 {
			return false
		}

		for _, token := range c.lineTokens[1:colonIndex] {
			if slices.Contains(types, token) {
				return true
			}
		}
		return false
	}
}

func and(predicates ...completionPredicate) completionPredicate {
	return func(c completionContext) bool {
		for _, predicate := range predicates {
			if !predicate(c) {
				return false
			}
		}
		return true
	}
}

func or(predicates ...completionPredicate) completionPredicate {
	return func(c completionContext) bool {
		for _, predicate := range predicates {
			if predicate(c) {
				return true
			}
		}
		return false
	}
}

func isLastTokenPunctuation(c completionContext) bool {
	if len(c.lineTokens) == 0 {
		return false
	}
	last := c.lastToken()
	return strings.ContainsAny(last, ";:,()[]{}.+-*/%<>=!&|^~?")
}

// makeKeywordItem creates a keyword completion entry with optional markdown documentation.
func makeKeywordItem(label string, predicate completionPredicate, doc string) completionItemPredicate {
	var documentation *lsp.MarkupContent
	if doc != "" {
		documentation = &lsp.MarkupContent{Kind: lsp.MarkupMarkdown, Value: doc}
	}

	return completionItemPredicate{
		predicate: predicate,
		item: lsp.CompletionItem{
			Label:         label,
			Kind:          lsp.CompletionKeyword,
			Documentation: documentation,
		},
	}
}
