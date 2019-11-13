package moviepoll

import (
    "html/template"
    "net/http"
    "fmt"
)

const TEMPLATE_DIR = "templates/"
const TEMPLATE_BASE = TEMPLATE_DIR + "base.html"

//var templates map[string]*template.Template

// templateDefs is static throughout the life of the server process
var templateDefs map[string]string = map[string]string{
    "movieinfo": "movie-info.html",
}

func (s *Server) registerTemplates() error {
    s.templates = make(map[string]*template.Template)

    for key, file := range templateDefs {
        t, err := template.ParseFiles(TEMPLATE_BASE, TEMPLATE_DIR + file)
        if err != nil {
            return fmt.Errorf("Error parsing template %s: %v", file, err)
        }

        s.templates[key] = t
    }
    return nil
}

func (s *Server) executeTemplate(w http.ResponseWriter, key string, data interface{}) error {
    t, ok := s.templates[key]
    if !ok {
        return fmt.Errorf("Template with key %q does not exist", key)
    }

    return t.Execute(w, data)
}
