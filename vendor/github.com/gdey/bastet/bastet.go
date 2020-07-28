// baste provides a quick way to apply a set of values to a set of templates, and
// concate the output it to an io.Writer
package bastet

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"text/template"
)

// ProcessingErr captures an error with processing
// the named template.
type ProcessingErr struct {
	Name string
	Err  error
}

// Error fulfills the Error interface
func (tpe *ProcessingErr) Error() string {
	if tpe == nil {
		return ""
	}
	return fmt.Sprintf("error processing template %s : %v", tpe.Name, tpe.Err)
}

// outputTemplate will process the reader as a template apply the values to it
// and write the result to the w writer.
// This function does not close the writer or the reader
func outputTemplate(w io.Writer, bt Template, values map[string]string) error {
	t, err := bt.Template()
	if err != nil {
		return err
	}
	return t.Execute(w, values)
}

// Template represents a template to fill out
type Template struct {
	Name   string
	Reader io.Reader
}

// Template returns the derived template.Template.
func (bt Template) Template() (*template.Template, error) {
	b, err := ioutil.ReadAll(bt.Reader)
	if err != nil {
		return nil, err
	}
	return template.New(bt.Name).Parse(string(b))
}

// Process will process each passed in template writing it's output to the given io.Writer.
func Process(w io.Writer, templates []Template, values map[string]string) error {
	for _, t := range templates {
		if err := outputTemplate(w, t, values); err != nil {
			return &ProcessingErr{
				Name: t.Name,
				Err:  err,
			}
		}
	}
	return nil
}

// ProcessFiles acts the same as Process but will first read in files from the provided filenames
func ProcessFiles(w io.Writer, filenames []string, values map[string]string) error {
	if len(filenames) == 0 {
		return errors.New("no files provided")
	}
	var tpls = make([]Template, len(filenames))

	for i, fname := range filenames {
		fh, err := os.Open(fname)
		if err != nil {
			return fmt.Errorf("error opening file %s for reading: %v", fname, err)
		}
		defer fh.Close()
		tpls[i].Name = "file:" + fname
		tpls[i].Reader = fh
	}
	return Process(w, tpls, values)
}
