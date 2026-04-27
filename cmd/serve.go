package cmd

import (
	"fmt"
	"os"

	"github.com/peacefixation/ssg/internal/config"
	"github.com/peacefixation/ssg/internal/datasource"
	"github.com/peacefixation/ssg/internal/server"
	"github.com/peacefixation/ssg/internal/site"
	"github.com/peacefixation/ssg/internal/watcher"
	"github.com/spf13/cobra"
)

var (
	servePort  int
	watchFiles bool
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Serve the static site for development",
	RunE:  runServe,
}

func init() {
	serveCmd.Flags().IntVarP(&servePort, "port", "p", 8080, "port to serve on")
	serveCmd.Flags().BoolVar(&watchFiles, "watch", false, "watch for changes and rebuild")
}

func runServe(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

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
		_, err = site.Build(currentCfg, registry, false)
		return err
	}

	fw, err := watcher.NewFileWatcher()
	if err != nil {
		return fmt.Errorf("creating file watcher: %w", err)
	}
	defer fw.Close()

	paths := []string{cfg.TemplateDir, cfg.ContentDir, cfgFile}
	w, err := watcher.New(fw, paths, rebuild)
	if err != nil {
		return fmt.Errorf("setting up watcher: %w", err)
	}

	fmt.Printf("Watching %v for changes...\n", paths)
	return w.Run(cmd.Context())
}
