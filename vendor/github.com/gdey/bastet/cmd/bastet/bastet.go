package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/gdey/bastet"
)

/*
Usage:
   bastet template.tpl name=value name=value name=value
   bastet -o stuff template.tpl  name=value name=value name=value
*/

var outputFilename string

func init() {
	flag.StringVar(&outputFilename, "output", "", "File to output to, stdout is default")
	flag.StringVar(&outputFilename, "o", "", "File to output to, stdout is default")
}

func usage() string {
	return fmt.Sprintf(`
Usage:
	%v [-o output.txt] template.tpl [name=value ...]
`,
		os.Args[0],
	)

}

// processArgs will take the remaining flag args and sort them into name=value pairs
// and template file names.
func processArgs() ([]string, map[string]string) {
	var templateFiles []string
	vals := make(map[string]string)
	args := flag.Args()

	for _, a := range args {
		a := strings.TrimSpace(a)
		if a == "" {
			continue
		}
		// need to figure out if this is a name=value or a template file.
		if strings.IndexByte(a, '=') == -1 {
			// this is a filename
			templateFiles = append(templateFiles, a)
			continue
		}

		parts := strings.SplitN(a, "=", 2)
		if len(parts) == 1 {
			key := strings.Replace(a, " ", "_", -1)
			vals[key] = ""
			continue
		}
		key := strings.Replace(parts[0], " ", "_", -1)
		vals[key] = parts[1]

	}

	return templateFiles, vals
}

func main() {
	flag.Parse()
	templateFiles, vals := processArgs()

	out := os.Stdout
	if outputFilename != "" {
		f, err := os.Create(outputFilename)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening file %v for writing.\n", outputFilename)
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		defer f.Close()
		out = f
	}

	var tpls []bastet.Template
	if len(templateFiles) == 0 {
		// we expect our template to come in from stdin.
		tpls = append(tpls, bastet.Template{Name: "stdin", Reader: os.Stdin})
	} else {
		for _, fname := range templateFiles {
			fh, err := os.Open(fname)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error opening file %v for reading.\n", fname)
				fmt.Fprintln(os.Stderr, err)
				os.Exit(4)
			}
			defer fh.Close()
			tpls = append(tpls, bastet.Template{Name: "file:" + fname, Reader: fh})
		}
	}

	if err := bastet.Process(out, tpls, vals); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(3)
	}
}
