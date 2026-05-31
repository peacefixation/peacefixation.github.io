package cmd

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/peacefixation/ssg/internal/config"
	"github.com/spf13/cobra"
)

var (
	importGRList   string
	importGRDryRun bool
	importGRForce  bool
)

var importGoodreadsCmd = &cobra.Command{
	Use:   "goodreads <csv-file>",
	Short: "Import books from a Goodreads library export CSV",
	Long: `Import books from a Goodreads library export CSV.

Rows where Exclusive Shelf is "read" or "currently-reading" are imported.
Each book becomes a JSON item file in the target list directory.
Status "currently-reading" is stored as "reading" in the item.

Existing files are detected by Goodreads Book ID in the filename and
skipped unless --force is set.

Example:
  ssg import goodreads ~/Downloads/goodreads_library_export.csv
  ssg import goodreads export.csv --list content/reading --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: runImportGoodreads,
}

func init() {
	importGoodreadsCmd.Flags().StringVar(&importGRList, "list", "", "target list directory (default: <contentDir>/books)")
	importGoodreadsCmd.Flags().BoolVar(&importGRDryRun, "dry-run", false, "print what would be created without writing files")
	importGoodreadsCmd.Flags().BoolVar(&importGRForce, "force", false, "overwrite existing files matched by Book ID")
	importCmd.AddCommand(importGoodreadsCmd)
}

func runImportGoodreads(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	listDir := importGRList
	if listDir == "" {
		listDir = filepath.Join(cfg.ContentDir, "books")
	}

	if !importGRDryRun {
		if err := os.MkdirAll(listDir, 0755); err != nil {
			return fmt.Errorf("creating list directory: %w", err)
		}
	}

	f, err := os.Open(args[0])
	if err != nil {
		return fmt.Errorf("opening CSV: %w", err)
	}
	defer f.Close()

	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		return fmt.Errorf("parsing CSV: %w", err)
	}
	if len(rows) < 2 {
		return fmt.Errorf("CSV has no data rows")
	}

	colIdx := buildColIndex(rows[0])

	required := []string{"Book Id", "Title", "Author", "My Rating", "Date Read", "Date Added", "Bookshelves", "Exclusive Shelf"}
	for _, col := range required {
		if _, ok := colIdx[col]; !ok {
			return fmt.Errorf("CSV is missing expected column %q", col)
		}
	}

	created, skipped := 0, 0

	for _, row := range rows[1:] {
		get := func(col string) string {
			i, ok := colIdx[col]
			if !ok || i >= len(row) {
				return ""
			}
			return strings.TrimSpace(row[i])
		}

		shelf := get("Exclusive Shelf")
		if shelf != "read" && shelf != "currently-reading" {
			continue
		}

		bookID := get("Book Id")
		if bookID == "" {
			continue
		}

		if !importGRForce {
			exists, err := grFileExists(listDir, bookID)
			if err != nil {
				return err
			}
			if exists {
				skipped++
				continue
			}
		}

		item, filename := buildGRItem(get, bookID, shelf)
		destPath := filepath.Join(listDir, filename)

		if importGRDryRun {
			fmt.Printf("would create %s\n", destPath)
			created++
			continue
		}

		out, err := json.MarshalIndent(item, "", "  ")
		if err != nil {
			return fmt.Errorf("marshalling item for book %s: %w", bookID, err)
		}
		if err := os.WriteFile(destPath, append(out, '\n'), 0644); err != nil {
			return fmt.Errorf("writing %s: %w", destPath, err)
		}
		fmt.Printf("created %s\n", destPath)
		created++
	}

	fmt.Printf("\n%d created, %d skipped (already exist)\n", created, skipped)
	return nil
}

// buildGRItem constructs the item map and filename for one CSV row.
func buildGRItem(get func(string) string, bookID, shelf string) (map[string]any, string) {
	item := map[string]any{
		"type":   "book",
		"url":    "https://www.goodreads.com/book/show/" + bookID,
		"title":  get("Title"),
		"author": get("Author"),
	}

	if shelf == "currently-reading" {
		item["status"] = "reading"
	} else {
		item["status"] = "read"
	}

	if r, err := strconv.Atoi(get("My Rating")); err == nil && r > 0 {
		item["my_rating"] = r
	}

	if dr := get("Date Read"); dr != "" {
		if t, err := time.Parse("2006/01/02", dr); err == nil {
			item["date_read"] = t.Format("2006-01-02")
		}
	}

	if bs := get("Bookshelves"); bs != "" {
		var tags []string
		for _, tag := range strings.Split(bs, ", ") {
			if tag = strings.TrimSpace(tag); tag != "" && tag != "currently-reading" {
				tags = append(tags, tag)
			}
		}
		if len(tags) > 0 {
			item["tags"] = tags
		}
	}

	// Timestamp: prefer Date Read, fall back to Date Added, then now.
	ts := grParseDate(get("Date Read"))
	if ts.IsZero() {
		ts = grParseDate(get("Date Added"))
	}
	if ts.IsZero() {
		ts = time.Now().UTC()
	}

	title := get("Title")
	slug := nonAlnum.ReplaceAllString(strings.ToLower(title), "-")
	slug = strings.Trim(slug, "-")
	filename := fmt.Sprintf("%s-gr%s-%s.json", ts.UTC().Format("20060102T150405Z"), bookID, slug)

	return item, filename
}

func grParseDate(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, _ := time.Parse("2006/01/02", s)
	return t
}

// grFileExists reports whether any file in dir contains "gr{bookID}" in its name.
func grFileExists(dir, bookID string) (bool, error) {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("reading list directory: %w", err)
	}
	marker := "gr" + bookID
	for _, entry := range entries {
		if strings.Contains(entry.Name(), marker) {
			return true, nil
		}
	}
	return false, nil
}

// buildColIndex maps column names to their index in a CSV header row.
func buildColIndex(header []string) map[string]int {
	m := make(map[string]int, len(header))
	for i, name := range header {
		m[strings.TrimSpace(name)] = i
	}
	return m
}
