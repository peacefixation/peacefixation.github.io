package cmd

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/peacefixation/ssg/internal/config"
	"github.com/peacefixation/ssg/internal/datasource"
	"github.com/peacefixation/ssg/internal/server"
	"github.com/peacefixation/ssg/internal/site"
	"github.com/peacefixation/ssg/internal/watcher"
	"github.com/spf13/cobra"
)

var (
	servePort   int
	watchFiles  bool
	serveDrafts bool
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Serve the static site for development",
	RunE:  runServe,
}

func init() {
	serveCmd.Flags().IntVarP(&servePort, "port", "p", 8080, "port to serve on")
	serveCmd.Flags().BoolVar(&watchFiles, "watch", false, "watch for changes and rebuild")
	serveCmd.Flags().BoolVar(&serveDrafts, "drafts", false, "include draft items")
}

// collectWatchPaths returns all directories under each dirPath, plus any plain
// file paths, so that inotify watches every subdirectory recursively.
func collectWatchPaths(paths ...string) ([]string, error) {
	var result []string
	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			// Skip paths that don't exist (e.g. cfgFile not yet created).
			continue
		}
		if !info.IsDir() {
			result = append(result, p)
			continue
		}
		err = filepath.WalkDir(p, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				result = append(result, path)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

func runServe(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	cfg.Drafts = serveDrafts
	cfg.YouTubeAPIKey = os.Getenv("YOUTUBE_DATA_API_KEY")

	if err := config.Validate(cfg); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	registry := datasource.DefaultRegistry()

	if _, err := site.Build(cfg, registry, false); err != nil {
		return fmt.Errorf("initial build failed: %w", err)
	}

	port := servePort
	if port == 8080 && cfg.Server.Port != 0 {
		port = cfg.Server.Port
	}

	srv := server.New(cfg.OutputDir, port)

	fmt.Printf("Serving %s at http://%s:%d\n", cfg.OutputDir, cfg.Server.Host, port)

	if !watchFiles {
		return srv.Start()
	}

	go func() {
		if err := srv.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		}
	}()

	rebuild := func() error {
		currentCfg, err := config.Load("")
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}
		currentCfg.Drafts = serveDrafts
		currentCfg.YouTubeAPIKey = os.Getenv("YOUTUBE_DATA_API_KEY")
		_, err = site.Build(currentCfg, registry, false)
		return err
	}

	fw, err := watcher.NewFileWatcher()
	if err != nil {
		return fmt.Errorf("creating file watcher: %w", err)
	}
	defer fw.Close()

	paths, err := collectWatchPaths(cfg.TemplateDir, cfg.ContentDir, cfg.ThemesDir, cfg.TypesDir, cfgFile)
	if err != nil {
		return fmt.Errorf("collecting watch paths: %w", err)
	}
	w, err := watcher.New(fw, paths, rebuild)
	if err != nil {
		return fmt.Errorf("setting up watcher: %w", err)
	}

	fmt.Printf("Watching %d paths for changes...\n", len(paths))
	return w.Run(cmd.Context())
}
