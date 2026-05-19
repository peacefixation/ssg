package cmd

import (
	"fmt"
	"time"

	"github.com/peacefixation/ssg/internal/config"
	"github.com/peacefixation/ssg/internal/datasource"
	"github.com/peacefixation/ssg/internal/site"
	"github.com/spf13/cobra"
)

var (
	outputDir   string
	cleanBuild  bool
	buildDrafts bool
	refreshOG   bool
)

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build the static site",
	RunE:  runBuild,
}

func init() {
	buildCmd.Flags().StringVarP(&outputDir, "output", "o", "", "output directory (overrides config)")
	buildCmd.Flags().BoolVar(&cleanBuild, "clean", false, "clean output directory before build")
	buildCmd.Flags().BoolVar(&buildDrafts, "drafts", false, "include draft items in the build")
	buildCmd.Flags().BoolVar(&refreshOG, "refresh-og", false, "bypass OG cache and re-fetch all opengraph items")
}

func runBuild(cmd *cobra.Command, args []string) error {
	start := time.Now()

	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if outputDir != "" {
		cfg.OutputDir = outputDir
	}
	cfg.Drafts = buildDrafts
	cfg.RefreshOG = refreshOG

	if err := config.Validate(cfg); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	registry := datasource.DefaultRegistry()

	count, err := site.Build(cfg, registry, cleanBuild)
	if err != nil {
		return fmt.Errorf("build failed: %w", err)
	}

	fmt.Printf("Built %d pages in %s → %s\n", count, time.Since(start).Round(time.Millisecond), cfg.OutputDir)
	return nil
}
