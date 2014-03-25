// main.go
package main

import (
  "github.com/codegangsta/martini"
  "github.com/martini-contrib/render"
)

// Defined here
type Content struct {
  Title string
  Body string
}

func main() {
  m := martini.Classic()
  // render html templates from templates directory
  m.Use(render.Renderer(render.Options{
    Extensions: []string{".tmpl", ".html"},
  }))

  data := Content{Title: "Hej", Body: "Här händer det mycket"}
  m.Get("/", func(r render.Render) {
    // Response code, title of template, input for template
    r.HTML(200, "hello", data)
  })

  m.Run()
}
