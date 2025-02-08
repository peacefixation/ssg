package build

import (
	"os"
	"path/filepath"

	"github.com/peacefixation/static-site-generator/internal/util"
)

func copyStaticFiles(outputDir, staticDir string) error {
	err := filepath.Walk(staticDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(staticDir, path)
		if err != nil {
			return err
		}

		if info.IsDir() {
			return util.CreateDir(filepath.Join(outputDir, relPath))
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		return os.WriteFile(filepath.Join(outputDir, relPath), content, 0644)
	})
	if err != nil {
		return ErrCopyStaticFile{Err: err}
	}

	return nil
}
