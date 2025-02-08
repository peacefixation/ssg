package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"

	"github.com/peacefixation/static-site-generator/internal/build"
	"github.com/peacefixation/static-site-generator/internal/watcher"
)

func main() {
	configDir := flag.String("config", "config", "Config directory path")
	contentDir := flag.String("content", "content", "Content directory path")
	staticDir := flag.String("static", "static", "Static directory path")
	templateDir := flag.String("templates", "templates", "Templates directory path")
	outputDir := flag.String("output", "output", "Output directory path")
	watch := flag.Bool("watch", false, "Watch for file changes")
	serve := flag.Bool("serve", false, "Serve the site")
	flag.Parse()

	buildConfig := build.Config{
		ContentDir:     *contentDir,
		TemplateDir:    *templateDir,
		OutputDir:      *outputDir,
		StaticDir:      *staticDir,
		SiteConfigPath: *configDir + "/site.yaml",
		LinkConfigPath: *configDir + "/links.yaml",
	}

	err := build.BuildSite(buildConfig)
	if err != nil {
		log.Fatal(err)
	}

	if *serve {
		fmt.Println("Serving the site at http://localhost:8080")
		go func() {
			listener, err := net.Listen("tcp", ":8080")
			if err != nil {
				log.Fatal(err)
			}

			err = http.Serve(listener, http.FileServer(http.Dir(*outputDir)))
			if err != nil {
				log.Fatal(err)
			}
		}()
	}

	if *watch {
		fmt.Println("Watching for changes...")
		watchLocations := []string{*configDir, *contentDir, *staticDir, *templateDir}
		err := watcher.Watch(watchLocations, func() error {
			return build.BuildSite(buildConfig)
		})
		if err != nil {
			log.Fatal(err)
		}
	}
}
