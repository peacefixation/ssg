package generate

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/peacefixation/static-site-generator/internal/file"
	"github.com/peacefixation/static-site-generator/internal/util"
)

type ErrCopyStaticFile struct{ error }

func (e ErrCopyStaticFile) Error() string {
	return fmt.Sprintf("error copying file: %v", e.error)
}

func (e ErrCopyStaticFile) Unwrap() error { return e.error }

func copyStaticFiles(outputDir, staticDir string, dirCreator file.DirCreator, fileReader file.FileReader, fileCreator file.FileCreator) error {
	err := filepath.Walk(staticDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// ignore .css files
		// if filepath.Ext(path) == ".css" {
		// 	return nil
		// }

		relPath, err := filepath.Rel(staticDir, path)
		if err != nil {
			return err
		}

		outPath := filepath.Join(outputDir, relPath)

		if info.IsDir() {
			return util.CreateDir(dirCreator, outPath)
		}

		content, err := fileReader.ReadFile(path)
		if err != nil {
			return err
		}

		writer, err := fileCreator.Create(outPath)
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
		return ErrCopyStaticFile{err}
	}

	return nil
}

func copyCSSFiles(outputDir, staticDir string, dirCreator file.DirCreator, fileReader file.FileReader, fileCreator file.FileCreator) error {
	cssFiles := make([]string, 0)

	err := filepath.Walk(staticDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// ignore non .css files
		if filepath.Ext(path) != ".css" {
			return nil
		}

		cssFiles = append(cssFiles, path)

		return nil
	})
	if err != nil {
		return ErrCopyStaticFile{err}
	}

	// create the output/css directory
	err = util.CreateDir(dirCreator, filepath.Join(outputDir, "css"))
	if err != nil {
		return err
	}

	// create a styles.css file in the output/css directory
	writer, err := fileCreator.Create(filepath.Join(outputDir, "css", "styles.css"))
	if err != nil {
		return err
	}
	defer writer.Close()

	for _, path := range cssFiles {
		content, err := fileReader.ReadFile(path)
		if err != nil {
			return err
		}

		_, err = writer.Write(content)
		if err != nil {
			return err
		}

		_, err = writer.Write([]byte("\n"))
		if err != nil {
			return err
		}
	}

	return nil
}
