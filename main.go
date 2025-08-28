package main

import (
	"bytes"
	_ "embed"
	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/kingpin/v2"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"html/template"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type Document struct {
	Title          string
	ExtraVariables map[string]string
	Stylesheets    []string
	Body           template.HTML
}

var (
	inputFiles     []string
	outputFilename string
	document       = Document{ExtraVariables: make(map[string]string)}

	//go:embed template.gohtml
	outputTemplateSource   string
	outputTemplateFilename string
	outputTemplate         *template.Template
)

func main() {
	kingpin.Flag("variable", "extra template variable").Short('e').StringMapVar(&document.ExtraVariables)
	kingpin.Flag("stylesheet", "stylesheet uri").Short('s').StringsVar(&document.Stylesheets)
	kingpin.Flag("output", "output filename").Short('o').StringVar(&outputFilename)
	kingpin.Flag("title", "document title").Short('t').StringVar(&document.Title)
	kingpin.Flag("template", "html template").Short('m').StringVar(&outputTemplateSource)
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

	var md = goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			highlighting.NewHighlighting(
				highlighting.WithFormatOptions(
					chromahtml.WithClasses(true),
				),
			),
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
