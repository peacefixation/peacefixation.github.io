package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/peacefixation/ssg/internal/config"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// Leaf-node flags (formerly ssg new item).
var (
	newNodeParent string
	newNodeType   string
)

// Branch-node flags (formerly ssg new list).
var (
	newNodeTitle        string
	newNodeTypes        string
	newNodeTemplate     string
	newNodeCardTemplate string
	newNodeSortBy       string
	newNodeSortOrder    string
	newNodeLimit        int
)

var newNodeCmd = &cobra.Command{
	Use:   "node",
	Short: "Create a new node (leaf or branch)",
	Long: `Create a new node.

Without a positional argument, creates a leaf node (content file):

  ssg new node [--parent <node>] [--type <type>] [key=value ...]

With a positional argument <name> and --title, creates a branch node
(directory + node.yaml):

  ssg new node <name> --title <title> [flags]

Examples:
  ssg new node --parent music --type youtube url=https://youtu.be/xyz title="My Song"
  ssg new node blog --title "Blog" --types post --sort-by date --sort-order desc`,
	Args: cobra.ArbitraryArgs,
	RunE: runNewNode,
}

func init() {
	// Leaf flags.
	newNodeCmd.Flags().StringVar(&newNodeParent, "parent", "", "parent node to add the leaf node to (defaults to root content directory)")
	newNodeCmd.Flags().StringVar(&newNodeType, "type", "", "node type for leaf node")

	// Branch flags.
	newNodeCmd.Flags().StringVar(&newNodeTitle, "title", "", "branch node title (required when creating a branch)")
	newNodeCmd.Flags().StringVar(&newNodeTypes, "types", "", "comma-separated node type allowlist (branch nodes)")
	newNodeCmd.Flags().StringVar(&newNodeTemplate, "template", "", "override page template")
	newNodeCmd.Flags().StringVar(&newNodeCardTemplate, "card-template", "", "override child card template")
	newNodeCmd.Flags().StringVar(&newNodeSortBy, "sort-by", "", "field to sort children by")
	newNodeCmd.Flags().StringVar(&newNodeSortOrder, "sort-order", "", "sort order: asc or desc")
	newNodeCmd.Flags().IntVar(&newNodeLimit, "limit", 0, "max children to render (0 = unlimited)")

	newCmd.AddCommand(newNodeCmd)
}

func runNewNode(cmd *cobra.Command, args []string) error {
	if newNodeSortOrder != "" && newNodeSortOrder != "asc" && newNodeSortOrder != "desc" {
		return fmt.Errorf("--sort-order must be asc or desc")
	}
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// Branch mode: --title is set, first positional arg is the directory name.
	if newNodeTitle != "" {
		if len(args) == 0 {
			return fmt.Errorf("a directory name is required when creating a branch node")
		}
		lc := branchNodeConfig{
			Title:        newNodeTitle,
			Template:     newNodeTemplate,
			CardTemplate: newNodeCardTemplate,
			SortBy:       newNodeSortBy,
			SortOrder:    newNodeSortOrder,
			Limit:        newNodeLimit,
		}
		if newNodeTypes != "" {
			for _, t := range strings.Split(newNodeTypes, ",") {
				if s := strings.TrimSpace(t); s != "" {
					lc.Types = append(lc.Types, s)
				}
			}
		}
		return createBranchNode(cfg, args[0], lc)
	}

	// Leaf mode: args are key=value pairs.
	return createLeafNode(cfg, newNodeParent, newNodeType, parseFields(args))
}

// createLeafNode creates a content file (leaf node).
func createLeafNode(cfg *config.SiteConfig, listName, typeName string, data map[string]string) error {
	var dir string
	if listName == "" {
		dir = cfg.ContentDir
	} else {
		meta, err := validateBranchNode(cfg.ContentDir, listName)
		if err != nil {
			return err
		}
		if err := validateType(meta, typeName, listName); err != nil {
			return err
		}
		dir = filepath.Join(cfg.ContentDir, listName)
	}

	format := "yaml"
	if typeName != "" {
		it, err := resolveNodeType(cfg.TypesDir, typeName)
		if err != nil {
			return err
		}
		if err := checkRequiredFields(it, data); err != nil {
			return err
		}
		format = it.Format
	}

	if format == "markdown" {
		return writeMarkdownItem(dir, data)
	}
	return writeYAMLItem(dir, typeName, data)
}

// nodeConfigFile returns the path to node.yaml in dir and true if found.
func nodeConfigFile(dir string) (string, bool) {
	p := filepath.Join(dir, "node.yaml")
	if _, err := os.Stat(p); err == nil {
		return p, true
	}
	return "", false
}

// writeYAMLItem writes a YAML content file into dir.
func writeYAMLItem(dir, typeName string, data map[string]string) error {
	item := make(map[string]any, len(data)+1)
	if typeName != "" {
		item["type"] = typeName
	}
	for k, v := range data {
		item[k] = v
	}
	out, err := yaml.Marshal(item)
	if err != nil {
		return fmt.Errorf("marshalling item: %w", err)
	}
	destPath := filepath.Join(dir, generateFilename(data["title"], ".yaml"))
	if err := os.WriteFile(destPath, out, 0644); err != nil {
		return fmt.Errorf("writing item: %w", err)
	}
	fmt.Printf("Created %s\n", destPath)
	return nil
}

// writeMarkdownItem writes a Markdown content file with YAML frontmatter into dir.
func writeMarkdownItem(dir string, data map[string]string) error {
	fm := make(map[string]any, len(data))
	for k, v := range data {
		fm[k] = v
	}
	fmBytes, err := yaml.Marshal(fm)
	if err != nil {
		return fmt.Errorf("marshalling frontmatter: %w", err)
	}
	content := "---\n" + string(fmBytes) + "---\n\n"
	destPath := filepath.Join(dir, generateFilename(data["title"], ".md"))
	if err := os.WriteFile(destPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("writing item: %w", err)
	}
	fmt.Printf("Created %s\n", destPath)
	return nil
}

// scanListExts is the set of extensions recognised as content files.
var scanListExts = map[string]bool{
	".md": true, ".markdown": true, ".json": true, ".yaml": true, ".yml": true,
}

// createBranchNode creates a branch node directory. It is the underlying implementation
// called by the "ssg new node <name>" branch mode.
func createBranchNode(cfg *config.SiteConfig, name string, lc branchNodeConfig) error {
	destDir := filepath.Join(cfg.ContentDir, name)

	if _, err := os.Stat(destDir); err == nil {
		return fmt.Errorf("directory %s already exists", destDir)
	}

	parentDir := filepath.Dir(destDir)
	listName := filepath.Base(destDir)

	// If the parent directory has no node config it may be a file item's sibling
	// container. The root content directory has no node config by convention and
	// is always valid.
	parentIsRoot := filepath.Clean(parentDir) == filepath.Clean(cfg.ContentDir)
	if !parentIsRoot {
		if _, found := nodeConfigFile(parentDir); !found {
			grandparentDir := filepath.Dir(parentDir)
			stem := filepath.Base(parentDir)
			itemPath, found := findFileItemByStem(grandparentDir, stem)
			if !found {
				if _, statErr := os.Stat(parentDir); os.IsNotExist(statErr) {
					return fmt.Errorf("parent directory %s does not exist", parentDir)
				}
				return fmt.Errorf("parent %q is not a node and no matching file item found", parentDir)
			}
			return createFileItemSubBranch(destDir, parentDir, listName, itemPath, lc)
		}
	}

	if cfgFile, found := nodeConfigFile(parentDir); found {
		if parent, err := readBranchConfig(cfgFile); err == nil {
			lc = mergeParentConfig(lc, parent)
		}
	}

	if err := os.Mkdir(destDir, 0755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	out, err := yaml.Marshal(lc)
	if err != nil {
		return fmt.Errorf("marshalling node config: %w", err)
	}

	nodeFile := filepath.Join(destDir, "node.yaml")
	if err := os.WriteFile(nodeFile, out, 0644); err != nil {
		return fmt.Errorf("writing node.yaml: %w", err)
	}

	fmt.Printf("Created %s\n", nodeFile)
	return nil
}

func createFileItemSubBranch(destDir, siblingDir, listName, itemPath string, lc branchNodeConfig) error {
	if err := os.MkdirAll(siblingDir, 0755); err != nil {
		return fmt.Errorf("creating sibling directory: %w", err)
	}
	if err := os.Mkdir(destDir, 0755); err != nil {
		return fmt.Errorf("creating sub-node directory: %w", err)
	}
	out, err := yaml.Marshal(lc)
	if err != nil {
		return fmt.Errorf("marshalling node config: %w", err)
	}
	nodeFile := filepath.Join(destDir, "node.yaml")
	if err := os.WriteFile(nodeFile, out, 0644); err != nil {
		return fmt.Errorf("writing node.yaml: %w", err)
	}
	fmt.Printf("Created %s\n", nodeFile)
	if err := appendListToFile(itemPath, listName); err != nil {
		return fmt.Errorf("updating %s: %w", itemPath, err)
	}
	fmt.Printf("Updated %s\n", itemPath)
	return nil
}

type branchNodeConfig struct {
	Title        string   `yaml:"title"`
	Types        []string `yaml:"types,omitempty"`
	Template     string   `yaml:"template,omitempty"`
	CardTemplate string   `yaml:"cardTemplate,omitempty"`
	SortBy       string   `yaml:"sortBy,omitempty"`
	SortOrder    string   `yaml:"sortOrder,omitempty"`
	Limit        int      `yaml:"limit,omitempty"`
}

func readBranchConfig(path string) (branchNodeConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return branchNodeConfig{}, err
	}
	var lc branchNodeConfig
	if err := yaml.Unmarshal(data, &lc); err != nil {
		return branchNodeConfig{}, err
	}
	return lc, nil
}

func mergeParentConfig(lc, parent branchNodeConfig) branchNodeConfig {
	if lc.CardTemplate == "" {
		lc.CardTemplate = parent.CardTemplate
	}
	if lc.Template == "" {
		lc.Template = parent.Template
	}
	if lc.SortBy == "" {
		lc.SortBy = parent.SortBy
	}
	if lc.SortOrder == "" {
		lc.SortOrder = parent.SortOrder
	}
	if lc.Limit == 0 {
		lc.Limit = parent.Limit
	}
	if lc.Types == nil {
		lc.Types = parent.Types
	}
	return lc
}

// --- list discovery ---

// fileItemMeta holds the fields from a content file that are relevant to list discovery.
type fileItemMeta struct {
	Title string   `yaml:"title" json:"title"`
	Lists []string `yaml:"lists" json:"lists"`
}

// readFileItemMeta reads title and lists from a content file (YAML, JSON, or Markdown).
func readFileItemMeta(path string) fileItemMeta {
	data, err := os.ReadFile(path)
	if err != nil {
		return fileItemMeta{}
	}
	var m fileItemMeta
	switch strings.ToLower(filepath.Ext(path)) {
	case ".json":
		_ = json.Unmarshal(data, &m)
	case ".md", ".markdown":
		if fm := extractFrontmatter(data); fm != nil {
			_ = yaml.Unmarshal(fm, &m)
		}
	default:
		_ = yaml.Unmarshal(data, &m)
	}
	return m
}

// extractFrontmatter returns the YAML block between leading --- delimiters, or nil.
func extractFrontmatter(data []byte) []byte {
	s := string(data)
	if !strings.HasPrefix(s, "---") {
		return nil
	}
	rest := s[3:]
	idx := strings.Index(rest, "\n---")
	if idx < 0 {
		return nil
	}
	return []byte(rest[:idx])
}

// stemFromPath returns the filename stem of path (base name without extension).
func stemFromPath(path string) string {
	base := filepath.Base(path)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

// findFileItemByStem looks in dir for a content file whose stem equals stem.
// Returns the full path and true if found.
func findFileItemByStem(dir, stem string) (string, bool) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", false
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".node.yaml") || name == "node.yaml" {
			continue
		}
		if stemFromPath(name) == stem && scanListExts[strings.ToLower(filepath.Ext(name))] {
			return filepath.Join(dir, name), true
		}
	}
	return "", false
}

// appendListToFile appends listName to the "lists" field of a content file in-place.
func appendListToFile(path, listName string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading %s: %w", path, err)
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".json":
		return appendListJSON(path, data, listName)
	case ".md", ".markdown":
		return appendListMarkdown(path, data, listName)
	default:
		return appendListYAML(path, data, listName)
	}
}

func appendListYAML(path string, data []byte, listName string) error {
	var m map[string]any
	if err := yaml.Unmarshal(data, &m); err != nil {
		return fmt.Errorf("parsing %s: %w", path, err)
	}
	if m == nil {
		m = make(map[string]any)
	}
	m["lists"] = append(toStringSlice(m["lists"]), listName)
	out, err := yaml.Marshal(m)
	if err != nil {
		return fmt.Errorf("marshalling %s: %w", path, err)
	}
	return os.WriteFile(path, out, 0644)
}

func appendListJSON(path string, data []byte, listName string) error {
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return fmt.Errorf("parsing %s: %w", path, err)
	}
	m["lists"] = append(toStringSlice(m["lists"]), listName)
	out, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling %s: %w", path, err)
	}
	return os.WriteFile(path, append(out, '\n'), 0644)
}

func appendListMarkdown(path string, data []byte, listName string) error {
	fm := extractFrontmatter(data)
	s := string(data)
	if fm == nil {
		newFM, err := yaml.Marshal(map[string]any{"lists": []string{listName}})
		if err != nil {
			return err
		}
		return os.WriteFile(path, []byte("---\n"+string(newFM)+"---\n\n"+s), 0644)
	}
	var fmMap map[string]any
	if err := yaml.Unmarshal(fm, &fmMap); err != nil {
		return fmt.Errorf("parsing frontmatter in %s: %w", path, err)
	}
	if fmMap == nil {
		fmMap = make(map[string]any)
	}
	fmMap["lists"] = append(toStringSlice(fmMap["lists"]), listName)
	newFM, err := yaml.Marshal(fmMap)
	if err != nil {
		return fmt.Errorf("marshalling frontmatter: %w", err)
	}
	rest := s[3:]
	idx := strings.Index(rest, "\n---")
	afterFM := rest[idx+4:]
	return os.WriteFile(path, []byte("---\n"+string(newFM)+"---"+afterFM), 0644)
}

// toStringSlice coerces the value stored under "lists" in a parsed map to []string.
func toStringSlice(v any) []string {
	switch val := v.(type) {
	case []string:
		return val
	case []any:
		out := make([]string, 0, len(val))
		for _, item := range val {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

// --- shared helpers ---

type branchDirMeta struct {
	title string
	types []string
}

func readBranchDirMeta(path string) branchDirMeta {
	data, err := os.ReadFile(path)
	if err != nil {
		return branchDirMeta{}
	}
	var raw struct {
		Title string   `yaml:"title"`
		Types []string `yaml:"types"`
	}
	_ = yaml.Unmarshal(data, &raw)
	return branchDirMeta{title: raw.Title, types: raw.Types}
}

type nodeType struct {
	typeName string
	Name     string      `yaml:"name"`
	Format   string      `yaml:"format"`
	Fields   []typeField `yaml:"fields"`
}

type typeField struct {
	Name     string `yaml:"name"`
	Required bool   `yaml:"required"`
}

func loadNodeTypes(typesDir string, allowedTypes []string) ([]nodeType, error) {
	entries, err := os.ReadDir(typesDir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	allowed := make(map[string]bool, len(allowedTypes))
	for _, t := range allowedTypes {
		allowed[t] = true
	}

	var types []nodeType
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".yaml" {
			continue
		}
		typeName := strings.TrimSuffix(entry.Name(), ".yaml")
		if len(allowed) > 0 && !allowed[typeName] {
			continue
		}
		data, err := os.ReadFile(filepath.Join(typesDir, entry.Name()))
		if err != nil {
			return nil, err
		}
		var t nodeType
		if err := yaml.Unmarshal(data, &t); err != nil {
			return nil, fmt.Errorf("parsing %s: %w", entry.Name(), err)
		}
		t.typeName = typeName
		types = append(types, t)
	}
	return types, nil
}

func validateBranchNode(contentDir, listName string) (branchDirMeta, error) {
	cfgFile, found := nodeConfigFile(filepath.Join(contentDir, listName))
	if !found {
		return branchDirMeta{}, fmt.Errorf("parent node %q not found (expected node.yaml in %s)",
			listName, filepath.Join(contentDir, listName))
	}
	return readBranchDirMeta(cfgFile), nil
}

func validateType(meta branchDirMeta, typeName, listName string) error {
	if len(meta.types) > 0 {
		found := false
		for _, t := range meta.types {
			if t == typeName {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("type %q is not allowed in node %q (allowed: %s)",
				typeName, listName, strings.Join(meta.types, ", "))
		}
	}
	return nil
}

func resolveNodeType(typesDir, typeName string) (nodeType, error) {
	types, err := loadNodeTypes(typesDir, []string{typeName})
	if err != nil {
		return nodeType{}, fmt.Errorf("loading node types: %w", err)
	}
	if len(types) == 0 {
		return nodeType{}, fmt.Errorf("node type %q not found in %s", typeName, typesDir)
	}
	return types[0], nil
}

func checkRequiredFields(it nodeType, data map[string]string) error {
	var missing []string
	for _, field := range it.Fields {
		if field.Required {
			if v, ok := data[field.Name]; !ok || strings.TrimSpace(v) == "" {
				missing = append(missing, field.Name)
			}
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required fields: %s", strings.Join(missing, ", "))
	}
	return nil
}

func parseFields(args []string) map[string]string {
	m := make(map[string]string, len(args))
	for _, arg := range args {
		k, v, ok := strings.Cut(arg, "=")
		if ok && k != "" {
			m[k] = v
		}
	}
	return m
}

var nonAlnum = regexp.MustCompile(`[^a-z0-9]+`)

func generateFilename(title, ext string) string {
	ts := time.Now().UTC().Format("20060102T150405Z")
	slug := nonAlnum.ReplaceAllString(strings.ToLower(title), "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		return ts + ext
	}
	return ts + "-" + slug + ext
}
