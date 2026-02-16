// Copyright (c) 2026 Adam Snyder <https://armsnyder.com> and contributors
// SPDX-License-Identifier: MIT

// Command gentables generates CSV files from the Godot shader reference
// documentation tables and function signatures. These tables drive completion,
// hover, and other data-driven language server features.
package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const (
	repoRootPath = "tutorials/shaders/shader_reference"
	repoRef      = "stable"

	outputRoot = "internal/app/tables"
	mixinRoot  = "internal/app/table_mixins"
)

func main() {
	if err := run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	tables := make(map[string]tableData)

	if err := traverseTree(repoRootPath, repoRef, func(path string, content []byte) error {
		return collectTables(path, content, tables)
	}); err != nil {
		return fmt.Errorf("traversing docs tree: %w", err)
	}

	if err := applyMixins(tables); err != nil {
		return fmt.Errorf("applying mixins: %w", err)
	}

	if err := prepareOutputDir(); err != nil {
		return fmt.Errorf("preparing output directory: %w", err)
	}

	if err := writeTables(tables); err != nil {
		return fmt.Errorf("writing tables: %w", err)
	}

	return nil
}

// prepareOutputDir clears and recreates the generated tables directory.
func prepareOutputDir() error {
	if err := os.RemoveAll(outputRoot); err != nil {
		return fmt.Errorf("remove %s: %w", outputRoot, err)
	}
	if err := os.MkdirAll(outputRoot, 0o750); err != nil {
		return fmt.Errorf("mkdir %s: %w", outputRoot, err)
	}
	return nil
}

// collectTables extracts tables from one docs file and stores them by output-relative path.
func collectTables(path string, content []byte, tables map[string]tableData) error {
	baseDir := strings.TrimSuffix(strings.TrimPrefix(path, repoRootPath+"/"), filepath.Ext(path))

	if err := traverseTables(content, func(header string, table tableData) error {
		rel := filepath.Join(baseDir, tableFileName(header)+".csv")
		tables[rel] = table
		return nil
	}); err != nil {
		return fmt.Errorf("traverse tables in %s: %w", path, err)
	}

	if strings.HasSuffix(path, "/shader_functions.rst") {
		for header, table := range scrapeShaderFunctionSignatures(content) {
			rel := filepath.Join(baseDir, tableFileName(header)+".csv")
			if _, ok := tables[rel]; ok {
				continue
			}
			tables[rel] = table
		}
	}

	return nil
}

// writeTables materializes all collected tables to CSV files in deterministic path order.
func writeTables(tables map[string]tableData) error {
	rels := make([]string, 0, len(tables))
	for rel := range tables {
		rels = append(rels, rel)
	}
	sort.Strings(rels)

	for _, rel := range rels {
		out := filepath.Join(outputRoot, rel)
		if err := os.MkdirAll(filepath.Dir(out), 0o750); err != nil {
			return fmt.Errorf("mkdir %s: %w", filepath.Dir(out), err)
		}
		slog.Info("Writing table", "path", out)
		if err := tableToCSV(tables[rel], out); err != nil {
			return fmt.Errorf("write table %s: %w", out, err)
		}
	}

	return nil
}

type treeEntry struct {
	Path        string `json:"path"`
	DownloadURL string `json:"download_url"`
	Type        string `json:"type"`
}

func traverseTree(path, ref string, fn func(path string, content []byte) error) error {
	slog.Info("Listing path", "path", path, "ref", ref)

	items, err := listTreeEntries(path, ref)
	if err != nil {
		return err
	}

	for _, item := range items {
		if err := processTreeEntry(item, ref, fn); err != nil {
			return err
		}
	}

	return nil
}

// listTreeEntries fetches one GitHub directory listing for the requested docs path.
func listTreeEntries(path, ref string) ([]treeEntry, error) {
	contents, err := get(fmt.Sprintf("https://api.github.com/repos/godotengine/godot-docs/contents/%s?ref=%s", path, ref))
	if err != nil {
		return nil, fmt.Errorf("list contents: %w", err)
	}

	var items []treeEntry
	if err := json.Unmarshal(contents, &items); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return items, nil
}

// processTreeEntry dispatches one tree item as either a file download or recursive directory walk.
func processTreeEntry(item treeEntry, ref string, fn func(path string, content []byte) error) error {
	switch item.Type {
	case "file":
		return processTreeFile(item, fn)
	case "dir":
		if err := traverseTree(item.Path, ref, fn); err != nil {
			return fmt.Errorf("traverse directory %s: %w", item.Path, err)
		}
	}

	return nil
}

// processTreeFile downloads and processes one .rst file entry.
func processTreeFile(item treeEntry, fn func(path string, content []byte) error) error {
	if filepath.Ext(item.Path) != ".rst" {
		return nil
	}
	if item.DownloadURL == "" {
		return fmt.Errorf("missing download url for %s", item.Path)
	}

	slog.Info("Downloading file", "path", item.Path)
	content, err := get(item.DownloadURL)
	if err != nil {
		return fmt.Errorf("download file %s: %w", item.Path, err)
	}
	if err := fn(item.Path, content); err != nil {
		return fmt.Errorf("process file %s: %w", item.Path, err)
	}

	return nil
}

func get(url string) ([]byte, error) {
	resp, err := http.Get(url) //nolint:gosec // URL is controlled by this generator.
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	return body, nil
}

type tableData struct {
	Header []string
	Rows   [][]string
}

func traverseTables(content []byte, fn func(header string, table tableData) error) error {
	lines := bytes.Split(content, []byte("\n"))
	currentHeader := ""
	var headerCandidate []byte
	seenHeaders := make(map[string]int)

	nextUniqueHeader := func(rawHeader string) string {
		header := strings.TrimSpace(rawHeader)
		if header == "" {
			header = "table"
		}
		if n := seenHeaders[header]; n > 0 {
			seenHeaders[header]++
			return fmt.Sprintf("%s_%d", header, n+1)
		}
		seenHeaders[header] = 1
		return header
	}

	for i := 0; i < len(lines); i++ {
		trimmed := bytes.TrimSpace(lines[i])

		if isSectionUnderline(trimmed) {
			currentHeader = string(bytes.TrimSpace(headerCandidate))
			continue
		}

		headerCandidate = lines[i]
		table, consumed, ok := parseAnyTable(lines[i:], trimmed)
		if !ok {
			continue
		}
		if err := fn(nextUniqueHeader(currentHeader), table); err != nil {
			return err
		}
		i += consumed - 1
	}

	return nil
}

// parseAnyTable attempts all supported table syntaxes at the current line and returns consumed lines.
func parseAnyTable(lines [][]byte, trimmed []byte) (tableData, int, bool) {
	if bytes.HasPrefix(lines[0], []byte("+--")) {
		table, rest := parseGridTable(lines)
		return table, len(lines) - len(rest), true
	}
	if bytes.HasPrefix(trimmed, []byte(".. list-table::")) {
		table, consumed, ok := parseListTable(lines)
		return table, consumed, ok
	}
	if bytes.HasPrefix(trimmed, []byte(".. csv-table::")) {
		table, consumed, ok := parseCSVTable(lines)
		return table, consumed, ok
	}
	return tableData{}, 0, false
}

func isSectionUnderline(line []byte) bool {
	if len(line) < 3 {
		return false
	}
	allowed := "=-~^\"#+`*:._"
	c := line[0]
	if !strings.ContainsRune(allowed, rune(c)) {
		return false
	}
	for _, b := range line[1:] {
		if b != c {
			return false
		}
	}
	return true
}

func parseGridTable(lines [][]byte) (table tableData, restLines [][]byte) {
	table.Header, lines = parseGridTableRow(lines)

	for len(lines) > 0 && bytes.HasPrefix(lines[0], []byte("+")) {
		var row []string
		row, lines = parseGridTableRow(lines)
		if len(row) == 0 {
			break
		}
		table.Rows = append(table.Rows, row)
	}

	return table, lines
}

func parseGridTableRow(lines [][]byte) (row []string, restLines [][]byte) {
	if len(lines) == 0 {
		return nil, lines
	}
	lines = lines[1:]
	var cells [][]byte

	for len(lines) > 0 && bytes.HasPrefix(lines[0], []byte("|")) {
		split := bytes.Split(lines[0][1:], []byte("|"))
		split = split[:len(split)-1]

		for j, part := range split {
			if j >= len(cells) {
				cells = append(cells, nil)
			}
			cells[j] = bytes.Join([][]byte{cells[j], bytes.TrimSpace(part)}, []byte(" "))
		}

		lines = lines[1:]
	}

	row = make([]string, len(cells))
	for i, cell := range cells {
		row[i] = string(bytes.TrimSpace(cell))
	}

	return row, lines
}

// parseListTable parses one rst list-table directive body into normalized tabular rows.
func parseListTable(lines [][]byte) (table tableData, consumed int, ok bool) {
	if len(lines) == 0 {
		return tableData{}, 0, false
	}
	i, headerRows := parseListDirectiveOptions(lines, 1)
	rows, i := parseListRows(lines, i)

	if len(rows) == 0 {
		return tableData{}, i, false
	}

	if headerRows > 0 && len(rows) >= headerRows {
		table.Header = normalizeRow(rows[0], len(rows[0]))
		table.Rows = rows[headerRows:]
	} else {
		width := maxRowWidth(rows)
		table.Header = syntheticHeader(width)
		table.Rows = rows
	}
	table = normalizeTable(table)
	return table, i, true
}

// parseListDirectiveOptions consumes list-table options and returns the next unread line and header row count.
func parseListDirectiveOptions(lines [][]byte, i int) (next, headerRows int) {
	for i < len(lines) {
		line := string(lines[i])
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			i++
			continue
		}
		if !isIndented(line) || !strings.HasPrefix(trimmed, ":") {
			break
		}
		if strings.HasPrefix(trimmed, ":header-rows:") {
			headerRows = parseIntSuffix(trimmed)
		}
		i++
	}
	return i, headerRows
}

// parseListRows consumes all list-table rows from the current offset.
func parseListRows(lines [][]byte, i int) (rows [][]string, next int) {
	for i < len(lines) {
		cells, next, ok := parseListRow(lines, i)
		if !ok {
			if !isIndented(string(lines[i])) {
				break
			}
			i = next
			continue
		}
		rows = append(rows, cells)
		i = next
	}
	return rows, i
}

// parseListRow parses one list-table row and returns the next unread offset.
func parseListRow(lines [][]byte, i int) (cells []string, next int, ok bool) {
	line := string(lines[i])
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return nil, i + 1, false
	}
	if !isIndented(line) {
		return nil, i, false
	}
	if !strings.HasPrefix(trimmed, "* -") {
		return nil, i + 1, false
	}

	cells = []string{strings.TrimSpace(strings.TrimPrefix(trimmed, "* -"))}
	i++
	for i < len(lines) {
		next := string(lines[i])
		nextTrimmed := strings.TrimSpace(next)
		if nextTrimmed == "" {
			i++
			continue
		}
		if !isIndented(next) || strings.HasPrefix(nextTrimmed, "* -") {
			break
		}
		if elem, ok := strings.CutPrefix(nextTrimmed, "-"); ok {
			cells = append(cells, strings.TrimSpace(elem))
			i++
			continue
		}
		cells[len(cells)-1] = strings.TrimSpace(cells[len(cells)-1] + " " + nextTrimmed)
		i++
	}

	return cells, i, true
}

// parseCSVTable parses one rst csv-table directive body into a table.
func parseCSVTable(lines [][]byte) (table tableData, consumed int, ok bool) {
	if len(lines) == 0 {
		return tableData{}, 0, false
	}

	i, headers, headerRows := parseCSVDirectiveOptions(lines, 1)
	block, i := collectIndentedDirectiveBlock(lines, i)
	r := csv.NewReader(strings.NewReader(strings.Join(block, "\n")))
	records, err := r.ReadAll()
	if err != nil || len(records) == 0 {
		return tableData{}, i, false
	}

	table = buildCSVDirectiveTable(records, headers, headerRows)
	table = normalizeTable(table)
	return table, i, true
}

// parseCSVDirectiveOptions consumes csv-table options and returns parsed header metadata.
func parseCSVDirectiveOptions(lines [][]byte, i int) (next int, headers []string, headerRows int) {
	headers = []string{}
	headerRows = 0
	for i < len(lines) {
		line := string(lines[i])
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			i++
			continue
		}
		if !isIndented(line) || !strings.HasPrefix(trimmed, ":") {
			break
		}
		if rawHeader, ok := strings.CutPrefix(trimmed, ":header:"); ok {
			csvHeader := strings.TrimSpace(rawHeader)
			r := csv.NewReader(strings.NewReader(csvHeader))
			parsed, err := r.Read()
			if err == nil {
				headers = parsed
			}
		}
		if strings.HasPrefix(trimmed, ":header-rows:") {
			headerRows = parseIntSuffix(trimmed)
		}
		i++
	}
	return i, headers, headerRows
}

// collectIndentedDirectiveBlock collects the indented payload block that follows a directive.
func collectIndentedDirectiveBlock(lines [][]byte, i int) (block []string, next int) {
	for i < len(lines) {
		line := string(lines[i])
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			block = append(block, "")
			i++
			continue
		}
		if !isIndented(line) {
			break
		}
		block = append(block, strings.TrimSpace(line))
		i++
	}
	return block, i
}

// buildCSVDirectiveTable selects header and row slices from parsed csv-table records.
func buildCSVDirectiveTable(records [][]string, headers []string, headerRows int) tableData {
	var table tableData
	switch {
	case len(headers) > 0:
		table.Header = headers
		table.Rows = records
	case headerRows > 0 && len(records) >= headerRows:
		table.Header = records[0]
		table.Rows = records[headerRows:]
	default:
		table.Header = records[0]
		table.Rows = records[1:]
	}
	return table
}

func parseIntSuffix(s string) int {
	parts := strings.Split(s, ":")
	n, err := strconv.Atoi(strings.TrimSpace(parts[len(parts)-1]))
	if err != nil {
		return 0
	}
	return n
}

func isIndented(s string) bool {
	return strings.HasPrefix(s, " ") || strings.HasPrefix(s, "\t")
}

func maxRowWidth(rows [][]string) int {
	maxWidth := 0
	for _, row := range rows {
		if len(row) > maxWidth {
			maxWidth = len(row)
		}
	}
	return maxWidth
}

func syntheticHeader(width int) []string {
	if width < 1 {
		return []string{"Value"}
	}
	header := make([]string, width)
	for i := range width {
		header[i] = fmt.Sprintf("Column %d", i+1)
	}
	return header
}

func normalizeRow(row []string, width int) []string {
	if width <= 0 {
		return []string{}
	}
	if len(row) >= width {
		return row[:width]
	}
	out := make([]string, width)
	copy(out, row)
	return out
}

func normalizeTable(t tableData) tableData {
	if len(t.Header) == 0 {
		width := maxRowWidth(t.Rows)
		t.Header = syntheticHeader(width)
	}
	width := len(t.Header)
	t.Header = normalizeRow(t.Header, width)
	for i := range t.Rows {
		t.Rows[i] = normalizeRow(t.Rows[i], width)
	}
	return t
}

// scrapeShaderFunctionSignatures extracts function signatures from shader_functions.rst prose sections.
func scrapeShaderFunctionSignatures(content []byte) map[string]tableData {
	lines := strings.Split(string(content), "\n")
	section := "functions"
	var candidate string

	sigPattern := regexp.MustCompile(`^\s*(.*?)\s*\*\*([A-Za-z_][A-Za-z0-9_]*)\*\*\s*\((.*)\)\s*$`)

	tables := map[string]tableData{}
	seen := map[string]map[string]struct{}{}

	for i := range lines {
		trimmed := strings.TrimSpace(lines[i])
		nextSection, wasSection := parseSectionHeader(trimmed, candidate)
		if wasSection {
			section = nextSection
			continue
		}
		candidate = lines[i]

		signature, ok := parseFunctionSignature(trimmed, sigPattern)
		if !ok {
			continue
		}

		header := normalizeFunctionHeader(section)
		if seenSignature(seen, header, signature) {
			continue
		}
		desc := findNextDocLine(lines, i+1)
		appendFunctionRow(tables, seen, header, signature, desc)
	}

	return tables
}

// parseSectionHeader updates the active section name when the current line is an rst underline marker.
func parseSectionHeader(trimmed, candidate string) (string, bool) {
	if !isSectionUnderline([]byte(trimmed)) {
		return "", false
	}
	if h := strings.TrimSpace(candidate); h != "" {
		return strings.ToLower(h), true
	}
	return "functions", true
}

// parseFunctionSignature parses a function signature line into a normalized signature string.
func parseFunctionSignature(line string, pattern *regexp.Regexp) (string, bool) {
	match := pattern.FindStringSubmatch(line)
	if match == nil {
		return "", false
	}

	signature := strings.TrimSpace(strings.TrimSpace(match[1]) + " " + match[2] + "(" + strings.TrimSpace(match[3]) + ")")
	if signature == "" || strings.Contains(signature, "|") {
		return "", false
	}
	return signature, true
}

// normalizeFunctionHeader resolves the fallback section name used for function buckets.
func normalizeFunctionHeader(header string) string {
	header = strings.TrimSpace(header)
	if header == "" {
		return "functions"
	}
	return header
}

// findNextDocLine returns the next non-empty, non-directive line after a signature.
func findNextDocLine(lines []string, start int) string {
	for j := start; j < len(lines); j++ {
		next := strings.TrimSpace(lines[j])
		if next == "" || strings.HasPrefix(next, "..") {
			continue
		}
		return cleanDocText(next)
	}
	return ""
}

// seenSignature reports whether a signature already exists in the per-section dedupe set.
func seenSignature(seen map[string]map[string]struct{}, header, signature string) bool {
	if _, ok := seen[header]; !ok {
		return false
	}
	_, ok := seen[header][signature]
	return ok
}

// appendFunctionRow initializes section storage when needed and appends one function row.
func appendFunctionRow(tables map[string]tableData, seen map[string]map[string]struct{}, header, signature, desc string) {
	if _, ok := tables[header]; !ok {
		tables[header] = tableData{Header: []string{"Function", "Description"}}
		seen[header] = map[string]struct{}{}
	}
	seen[header][signature] = struct{}{}
	t := tables[header]
	t.Rows = append(t.Rows, []string{signature, desc})
	tables[header] = t
}

func cleanDocText(s string) string {
	s = strings.TrimSpace(strings.TrimLeft(s, "-* "))
	s = strings.ReplaceAll(s, "**", "")
	s = strings.ReplaceAll(s, "``", "")
	return strings.TrimSpace(s)
}

// applyMixins overlays local CSV mixins onto generated tables using first-column key upserts.
func applyMixins(tables map[string]tableData) error {
	if err := ensureMixinRootDir(); err != nil {
		return err
	}

	return filepath.WalkDir(mixinRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Ext(path) != ".csv" {
			return nil
		}
		return applyMixinFile(tables, path)
	})
}

// ensureMixinRootDir validates mixinRoot when present and treats missing directory as no-op.
func ensureMixinRootDir() error {
	info, err := os.Stat(mixinRoot)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("stat mixin root: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("mixin root is not a directory: %s", mixinRoot)
	}
	return nil
}

// applyMixinFile applies one mixin CSV file to the in-memory table set.
func applyMixinFile(tables map[string]tableData, path string) error {
	rel, err := filepath.Rel(mixinRoot, path)
	if err != nil {
		return fmt.Errorf("relative path for mixin %s: %w", path, err)
	}

	slog.Info("Applying mixin", "path", path, "table", rel)
	mixinTable, err := readCSVTable(path)
	if err != nil {
		return fmt.Errorf("read mixin %s: %w", path, err)
	}

	base, ok := tables[rel]
	if !ok {
		tables[rel] = normalizeTable(mixinTable)
		return nil
	}

	merged, err := mergeTables(base, mixinTable)
	if err != nil {
		return fmt.Errorf("merge mixin %s: %w", path, err)
	}
	tables[rel] = merged
	return nil
}

func readCSVTable(path string) (tableData, error) {
	f, err := os.Open(path) //nolint:gosec // Path is under local workspace.
	if err != nil {
		return tableData{}, err
	}
	defer func() {
		_ = f.Close()
	}()

	r := csv.NewReader(f)
	records, err := r.ReadAll()
	if err != nil {
		return tableData{}, err
	}
	if len(records) == 0 {
		return tableData{}, errors.New("empty csv")
	}
	return tableData{Header: records[0], Rows: records[1:]}, nil
}

func mergeTables(base, mixin tableData) (tableData, error) {
	base = normalizeTable(base)
	mixin = normalizeTable(mixin)

	if !equalStringsFold(mixin.Header, base.Header) {
		return tableData{}, fmt.Errorf("header mismatch, expected %v got %v", base.Header, mixin.Header)
	}

	out := tableData{Header: base.Header}
	out.Rows = append(out.Rows, base.Rows...)
	indexByKey := make(map[string]int, len(out.Rows))
	for i, row := range out.Rows {
		if len(row) == 0 {
			continue
		}
		key := strings.TrimSpace(row[0])
		if key != "" {
			indexByKey[key] = i
		}
	}
	for _, mrow := range mixin.Rows {
		if len(mrow) == 0 {
			continue
		}
		row := normalizeRow(mrow, len(base.Header))
		key := strings.TrimSpace(row[0])
		if key == "" {
			continue
		}
		if idx, ok := indexByKey[key]; ok {
			out.Rows[idx] = row
			continue
		}
		indexByKey[key] = len(out.Rows)
		out.Rows = append(out.Rows, row)
	}
	return out, nil
}

func equalStringsFold(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !strings.EqualFold(strings.TrimSpace(a[i]), strings.TrimSpace(b[i])) {
			return false
		}
	}
	return true
}

// tableFileName normalizes a section header into a stable snake_case CSV basename.
func tableFileName(header string) string {
	header = strings.ToLower(strings.TrimSpace(header))
	if header == "" {
		return "table"
	}

	parts := strings.Fields(header)
	header = strings.Join(parts, "_")

	var b strings.Builder
	for _, r := range header {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '_':
			b.WriteRune(r)
		}
	}

	name := strings.Trim(b.String(), "_")
	if name == "" {
		return "table"
	}
	return name
}

// tableToCSV writes one table to disk preserving explicit cell empties in each row.
func tableToCSV(table tableData, path string) (terr error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600) //nolint:gosec // Path variable
	if err != nil {
		return err
	}
	defer func() {
		if err := f.Close(); err != nil && terr == nil {
			terr = err
		}
	}()

	if len(table.Header) == 0 {
		table = normalizeTable(table)
	}

	w := csv.NewWriter(f)
	if err := w.Write(table.Header); err != nil {
		return fmt.Errorf("write header: %w", err)
	}

	for i, row := range table.Rows {
		if err := w.Write(row); err != nil {
			return fmt.Errorf("write row %d: %w", i, err)
		}
	}

	w.Flush()
	return w.Error()
}
