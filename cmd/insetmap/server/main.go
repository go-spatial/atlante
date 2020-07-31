package main

import (
	"context"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/go-chi/chi"
	"github.com/go-spatial/atlante/insetmap"
	"github.com/go-spatial/atlante/internal/env"
	"github.com/jackc/pgx/v4/pgxpool"
)

const (
	EntryListingTemplate = `
 <!DOCTYPE html>
<html>
<body>
Currently available Entries are:
<dl>
{{ $mdgid := .Mdgid }}
{{range .Items}}
<dt><a href="/{{.Name}}/mdgid/{{$mdgid}}">{{if .IsDefault}}<b>{{end}}{{.Name}}{{if .IsDefault}}</b>{{end}}</a></dt>
<dd>
<p>{{.Description}}</p>
{{ if .CSSMap }} 
<p>CSSKeys:</p>
<dl>
{{ range $name, $cssmap := .CSSMap}}
<dt>{{$name}}</dt>
<dd>{{$cssmap.Desc}}</dd>
{{end}}
</dl>
{{end}}
</dd>
{{end}}
</dl>
</body>
</html>

`
)

var (
	mdgid      = flag.String("mdgid", "V795G25492", "default mdgid for listing example")
	configFile = flag.String("config", "config.toml", "config file")
	InsetMaps  map[string]InsetEntry
	DefaultMap string
)

type InsetEntry struct {
	Description string
	*insetmap.Inset
}
type ConfigEntry struct {
	DBConnStr env.String `toml:"database"`
	Desc      env.String `toml:"desc"`
	insetmap.Config
}

type Config struct {
	Addr          env.String             `toml:"address"`
	DefaultEntry  env.String             `toml:"default"`
	CSSDir        env.String             `toml:"css_dir"`
	CSSDefault    env.String             `toml:"css_default"`
	EntryListHTML string                 `toml:"entry_list_html"`
	Entries       map[string]ConfigEntry `toml:"entry"`
}

type ListingDataItem struct {
	Name        string
	Description template.HTML
	IsDefault   bool
	CSSMap      insetmap.CSSMap
	CSSDefault  string
}
type ListingData struct {
	Mdgid string
	Items []ListingDataItem
}

func BuildStaticListing(entryListHTMLTpl string, data ListingData) (string, error) {
	var out strings.Builder

	if entryListHTMLTpl == "" {
		entryListHTMLTpl = EntryListingTemplate
	}
	entryListTemplate, err := template.New("entrylist").Parse(entryListHTMLTpl)
	if err != nil {
		return "", err
	}

	err = entryListTemplate.Execute(&out, data)
	if err != nil {
		return "", err
	}
	return out.String(), nil
}

func main() {
	flag.Parse()
	var (
		config  Config
		gCSSMap insetmap.CSSMap
	)
	if _, err := toml.DecodeFile(*configFile, &config); err != nil {
		fmt.Fprintf(os.Stderr, "Unable to parse config(%v): %v\n", *configFile, err)
		os.Exit(1)
		return
	}
	gCSSDir := string(config.CSSDir)
	gCSSDefault := string(config.CSSDefault)
	if gCSSDir != "" {
		gCSSMap = make(insetmap.CSSMap)
		if err := gCSSMap.GetStyleSheets(gCSSDir); err != nil {
			fmt.Fprintf(os.Stderr, "Unable to parse cssDir: %v: %v\n", gCSSDir, err)
			os.Exit(1)
			return
		}
	}

	DefaultMap = string(config.DefaultEntry)
	InsetMaps = make(map[string]InsetEntry)

	lsData := ListingData{
		Mdgid: *mdgid,
	}

	for name, entry := range config.Entries {
		if DefaultMap == "" {
			// Use the first map as the default
			DefaultMap = name
		}

		conn, err := pgxpool.Connect(context.Background(), string(entry.DBConnStr))
		if err != nil {
			fmt.Fprintf(os.Stderr, "[%v] Unable to connect to database: %v\n", name, err)
			os.Exit(1)
		}

		defer conn.Close()
		imap, err := insetmap.New(conn, entry.Config, gCSSDir, gCSSMap, gCSSDefault)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[%v] Unable to parse entry: %v", name, err)
			os.Exit(1)
		}
		InsetMaps[name] = InsetEntry{
			Description: string(entry.Desc),
			Inset:       imap,
		}
		lsData.Items = append(lsData.Items, ListingDataItem{
			Name:        name,
			Description: template.HTML(entry.Desc),
			IsDefault:   DefaultMap == name,
			CSSMap:      imap.CSSMap,
			CSSDefault:  imap.CSSDefault,
		})

	}

	entrylisting, err := BuildStaticListing(config.EntryListHTML, lsData)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to parse entry list html: %v\n", err)
		os.Exit(1)
	}

	// Set up routes

	r := chi.NewRouter()
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(entrylisting))
	})
	r.Get("/{entry}/mdgid/{mdgid}", func(w http.ResponseWriter, r *http.Request) {
		entryName := chi.URLParam(r, "entry")
		entry, ok := InsetMaps[entryName]
		if !ok {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}
		mdgid := chi.URLParam(r, "mdgid")
		if mdgid == "" {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		cssKey := strings.TrimSpace(r.URL.Query().Get("css"))

		inset, err := entry.For(r.Context(), mdgid, cssKey)
		if err != nil {
			log.Printf("for mdgid(%v) got error: %v", mdgid, err)
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}
		partial := false
		{
			partialstr := strings.TrimSpace(r.URL.Query().Get("partial"))

			if partialstr != "" {
				partial, _ = strconv.ParseBool(partialstr)
			}

		}

		var attr strings.Builder
		{
			str := strings.TrimSpace(r.URL.Query().Get("w"))

			if str != "" {
				fmt.Fprintf(&attr, " width=\"%v\"", str)
			}
			str = strings.TrimSpace(r.URL.Query().Get("h"))

			if str != "" {
				fmt.Fprintf(&attr, " height=\"%v\"", str)
			}
		}

		svg, err := inset.AsSVG(partial, attr.String())
		if err != nil {
			log.Printf("while generating svg for  mdgid(%v) got error: %v", mdgid, err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", insetmap.SVGMime)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(svg))
	})

	// get address
	addr := strings.TrimSpace(string(config.Addr))
	if addr == "" {
		addr = ":0"
	}
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to listen on %v", addr)
		os.Exit(1)
	}

	log.Println("Default Map is:", DefaultMap)
	log.Println("Listening on port:", listener.Addr().(*net.TCPAddr).Port)
	panic(http.Serve(listener, r))

}
