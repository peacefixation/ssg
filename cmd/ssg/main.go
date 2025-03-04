package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/peacefixation/static-site-generator/internal/file"
	"github.com/peacefixation/static-site-generator/internal/generate"
	"github.com/peacefixation/static-site-generator/internal/opengraph"
	"github.com/peacefixation/static-site-generator/internal/parse"
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
	links := flag.Bool("links", false, "Fetch OpenGraph data for links and store it in links-og.yaml")
	flag.Parse()

	dirCreator := file.OSDirCreator{}
	fileCreator := file.OSFileCreator{}
	fileReader := file.OSFileReader{}

	// parse the site config
	siteConfigContent, err := fileReader.ReadFile(*configDir + "/site.yaml")
	if err != nil {
		log.Fatal(err)
	}

	siteConfig, err := parse.ParseSiteConfig(siteConfigContent)
	if err != nil {
		log.Fatal(err)
	}

	// parse the links config
	linkContent, err := fileReader.ReadFile(*configDir + "/links.yaml")
	if err != nil {
		log.Fatal(err)
	}

	linkData, err := parse.ParseLinks(linkContent)
	if err != nil {
		log.Fatal(err)
	}

	// if the links flag is given, update links.yaml and stop // TODO: this is a bit clunky
	if *links {
		// fetch OpenGraph data for links and update links.yaml
		opengraph.Fetch(opengraph.DefaultHTTPClient(), siteConfig.OpenGraphUserAgent, linkData.Links)
		err = parse.WriteLinks(linkData, fileCreator, *configDir+"/links-og.yaml")
		if err != nil {
			log.Fatal(err)
		}

		// remove old links.yaml
		err = os.Remove(*configDir + "/links.yaml")
		if err != nil {
			log.Fatal(err)
		}

		// rename links-og.yaml to links.yaml
		err = os.Rename(*configDir+"/links-og.yaml", *configDir+"/links.yaml")
		if err != nil {
			log.Fatal(err)
		}

		return // stop here
	}

	generator := generate.Generator{
		ContentDir:  *contentDir,
		TemplateDir: *templateDir,
		OutputDir:   *outputDir,
		StaticDir:   *staticDir,
		Title:       siteConfig.Title,
		ChromaStyle: siteConfig.SyntaxHighlightStyle,
		Links:       linkData.Links,
		DirCreator:  dirCreator,
		FileReader:  fileReader,
		FileCreator: fileCreator,
	}

	err = generator.GenerateSite()
	if err != nil {
		log.Fatal(err)
	}

	// serve the site
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

	// watch the file system for changes and rebuild the site
	if *watch {
		fmt.Println("Watching for changes...")
		watchLocations := []string{*configDir, *contentDir, *staticDir, *templateDir}
		err := watcher.Watch(watchLocations, func() error {
			// return build.BuildSite(buildConfig, dirCreator, fileReader, fileCreator)
			return generator.GenerateSite()
		})
		if err != nil {
			log.Fatal(err)
		}
	}
}
