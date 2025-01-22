package handler

import (
	"embed"
	"html/template"
	"net/http"
	"strings"
)

//go:embed homepage/*.html
var templates embed.FS

type handler struct{}

type Info struct {
	Title       string
	Description string
	LicenseUrl  string
}

type API struct {
	Method      string
	Path        string
	Description string
}

type Section struct {
	Name string
	APIs []API
}

type PageData struct {
	Info     Info
	Sections []Section
}

func NewHandler() *http.ServeMux {
	h := handler{}
	return registerRoutes(h)
}

func (h handler) homepageHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	tmpl, err := template.New("index.html").Funcs(template.FuncMap{
		"methodClass": func(method string) string {
			return "method-" + strings.ToLower(method)
		},
		"apiClass": func(method string) string {
			return "api-" + strings.ToLower(method)
		},
	}).ParseFS(templates, "homepage/index.html", "homepage/apis.html", "homepage/styles.html")
	if err != nil {
		http.Error(w, "Unable to load template", http.StatusInternalServerError)
		return
	}

	data := PageData{
		Info: Info{
			Title:       "API Homepage",
			Description: "This is a custom implementation of a Swagger-like UI.",
			LicenseUrl:  "https://opensource.org/licenses/MIT",
		},
		Sections: []Section{
			{
				Name: "Section 1",
				APIs: []API{
					{Method: "GET", Path: "/api/v1/resource1", Description: "Resource 1 endpoint"},
					{Method: "POST", Path: "/api/v1/resource2", Description: "Resource 2 endpoint"},
					{Method: "PUT", Path: "/api/v1/resource3", Description: "Resource 3 endpoint"},
				},
			},
			{
				Name: "Section 2",
				APIs: []API{
					{Method: "DELETE", Path: "/api/v1/resource4", Description: "Resource 4 endpoint"},
					{Method: "PATCH", Path: "/api/v1/resource5-wery-wery-long", Description: "Resource 5 endpoint"},
					{Method: "GET", Path: "/api/v1/resource6", Description: "Resource 6 endpoint"},
				},
			},
			{
				Name: "Section 3",
				APIs: []API{
					{Method: "POST", Path: "/api/v1/resource7", Description: "Resource 7 endpoint"},
					{Method: "PUT", Path: "/api/v1/resource8", Description: "Resource 8 endpoint"},
					{Method: "DELETE", Path: "/api/v1/resource9", Description: "Resource 9 endpoint"},
				},
			},
		},
	}
	if err := tmpl.ExecuteTemplate(w, "index.html", data); err != nil {
		http.Error(w, "Unable to render template", http.StatusInternalServerError)
	}
}
