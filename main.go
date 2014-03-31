// main.go
package main

import (
	"database/sql"
	"github.com/codegangsta/martini"
	"github.com/codegangsta/martini-contrib/binding"
	"github.com/coopernurse/gorp"
	"github.com/martini-contrib/render"
	_ "github.com/mattn/go-sqlite3"
	_"fmt"
)

// Database
type Kitchen struct {
	Item   string 	`form:"Item"`
	Amount int 		`form:"Amount"`
}

func main() {
	m := martini.Classic()
	// render html templates from templates directory
	m.Use(render.Renderer(render.Options{
		Extensions: []string{".tmpl", ".html"},
	}))

	m.Use(dbHandler())

	m.Get("/", func(r render.Render) {
		data := map[string]interface{}{"title": "Hej", "body": "Här händer det mycket", "kitchen": nil}
		// Response code, title of template, input for template
		r.HTML(200, "item_admin", data)
	})

	m.Get("/items", func(r render.Render, db *gorp.DbMap) {
		var kitchens []Kitchen // Where to save the DB SELECT
		_, _ = db.Select(&kitchens, "SELECT * FROM kitchen") // Query
		data := map[string]interface{}{"title": "Hej", "body": "Här händer det mycket", "kitchen": kitchens}
		r.HTML(200, "list", data)
	})

	// binding.Form = magic to bind a struct to elements from a form
	m.Post("/items", binding.Form(Kitchen{}), func(kitchen Kitchen, r render.Render, db *gorp.DbMap) {
		db.Insert(&kitchen) // Insert into DB
		kitchens, _ := db.Select(Kitchen{}, "SELECT * FROM kitchen") // Possibly to save into struct right away
		data := map[string]interface{}{"title": "Hej", "body": "Här händer det mycket", "kitchen": kitchens}
		r.HTML(200, "list", data)
	})
	m.Run()
}

// Database
func initDb() *gorp.DbMap {
	// connect to db using standard Go database/sql API
	// use whatever database/sql driver you wish
	db, _ := sql.Open("sqlite3", "/tmp/example_db.db")
	//checkErr(err, "sql.Open failed")

	// construct a gorp DbMap
	dbmap := &gorp.DbMap{Db: db, Dialect: gorp.SqliteDialect{}}

	// add a table, setting the table name to 'posts' and
	// specifying that the Id property is an auto incrementing PK
	dbmap.AddTableWithName(Kitchen{}, "kitchen").SetKeys(false, "Item")

	// create the table. in a production system you'd generally
	// use a migration tool, or create the tables via scripts
	_ = dbmap.CreateTablesIfNotExists()
	//checkErr(err, "Create tables failed")

	return dbmap
}

// Database middleware
func dbHandler() martini.Handler {
	// Return a martini.Handler to be called for every request
	return func(c martini.Context) {
		dbmap := initDb()
		c.Map(dbmap)
		defer dbmap.Db.Close()
		c.Next()
	}
}
