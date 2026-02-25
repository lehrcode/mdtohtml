package main

import (
	"bytes"
	_ "embed"
	"html/template"
	"log"
	"os"
	"path/filepath"
	"strings"

	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/kingpin/v2"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"go.abhg.dev/goldmark/toc"
)

type Document struct {
	Title         string
	Variables     map[string]string
	Stylesheets   []string
	ScriptModules []string
	Body          template.HTML
}

var (
	inputFiles     []string
	outputFilename string
	document       = Document{Variables: make(map[string]string)}

	//go:embed template.gohtml
	outputTemplateSource   string
	outputTemplateFilename string
	outputTemplate         *template.Template
	generateTOC            bool
	titleOfTOC             string
	minDepthOfTOC          int
	maxDepthOfTOC          int
)

func main() {
	kingpin.Flag("variable", "extra template variable").Short('e').StringMapVar(&document.Variables)
	kingpin.Flag("stylesheet", "stylesheet uri").Short('s').StringsVar(&document.Stylesheets)
	kingpin.Flag("script-module", "JavaScript module uri").Short('j').StringsVar(&document.ScriptModules)
	kingpin.Flag("output", "output filename").Short('o').StringVar(&outputFilename)
	kingpin.Flag("title", "document title").Short('t').StringVar(&document.Title)
	kingpin.Flag("template", "html template").Short('m').StringVar(&outputTemplateSource)
	kingpin.Flag("toc", "generate table-of-contents").BoolVar(&generateTOC)
	kingpin.Flag("toc-title", "table-of-contents title").Default("Contents").StringVar(&titleOfTOC)
	kingpin.Flag("toc-min-depth", "minimum headline depth included in table-of-contents").IntVar(&minDepthOfTOC)
	kingpin.Flag("toc-max-depth", "maximum headline depth included in table-of-contents").IntVar(&maxDepthOfTOC)
	kingpin.Arg("md", "markdown filename").Required().ExistingFilesVar(&inputFiles)
	kingpin.Parse()

	if outputTemplateFilename != "" {
		log.Printf("Output template filename: %s\n", outputTemplateFilename)
		if t, err := template.ParseFiles(outputTemplateFilename); err != nil {
			log.Fatal(err)
		} else {
			outputTemplate = t
		}
	} else {
		outputTemplate = template.Must(template.New("template.gohtml").Parse(outputTemplateSource))
	}

	var extensions = []goldmark.Extender{
		extension.GFM,
		highlighting.NewHighlighting(
			highlighting.WithFormatOptions(
				chromahtml.WithClasses(true),
			),
		),
	}

	if generateTOC {
		extensions = append(extensions, &toc.Extender{
			Title:    titleOfTOC,
			TitleID:  "toc-title",
			ListID:   "toc-list",
			Compact:  true,
			MinDepth: minDepthOfTOC,
			MaxDepth: maxDepthOfTOC,
		})
	}

	var md = goldmark.New(
		goldmark.WithExtensions(
			extensions...,
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			html.WithUnsafe(),
		),
	)

	var convertedHtml bytes.Buffer
	for _, inputFilename := range inputFiles {
		if inputBytes, err := os.ReadFile(inputFilename); err == nil {
			if err := md.Convert(inputBytes, &convertedHtml); err != nil {
				log.Fatal(err)
			}
		}
	}

	var finalHtml bytes.Buffer
	document.Body = template.HTML(convertedHtml.String())
	if err := outputTemplate.Execute(&finalHtml, document); err != nil {
		log.Fatal(err)
	}
	if outputFilename == "" {
		outputFilename = strings.TrimSuffix(filepath.Base(inputFiles[0]), filepath.Ext(inputFiles[0])) + ".html"
	}

	if err := os.WriteFile(outputFilename, finalHtml.Bytes(), 0644); err != nil {
		log.Fatal(err)
	}
}
