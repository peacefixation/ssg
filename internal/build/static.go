package build

import (
	"os"
	"path/filepath"

	"github.com/peacefixation/static-site-generator/internal/file"
	"github.com/peacefixation/static-site-generator/internal/util"
)

func copyStaticFiles(outputDir, staticDir string, fileCreator file.FileCreator) error {
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

		writer, err := fileCreator.Create(filepath.Join(outputDir, relPath))
		if err != nil {
			return err
		}
		defer writer.Close()

		_, err = writer.Write(content)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return ErrCopyStaticFile{Err: err}
	}

	return nil
}
