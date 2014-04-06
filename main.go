// main.go
package main

import (
	"database/sql"
	"github.com/codegangsta/martini"
	"github.com/codegangsta/martini-contrib/binding"
	"github.com/coopernurse/gorp"
	"github.com/martini-contrib/render"
	_"github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	_"strconv"
	"fmt"
)

// Database
type Kitchen struct {
	Item   string 	`form:"Item"`
	Amount sql.NullFloat64 		`form:"Amount"`
}

type KitchenForm struct {
	Item   string 	`form:"Item"`
	Amount float64 `form:"Amount"`
	Unit string `form:"Unit"`
}

type Food struct {
	Name string 	`form:"Item"`
	Unit string 	`form:"Amount"`
}

type Recipe struct {
	Name 		string 	`form:"Item"`
	Type 		string 	`form:"Type"`
	Description string 	`form:"Description"`
	Possible	string
}

type RecipeIngredients struct {
	Name 		string 	`form:"Item"`
	FoodName 	string 	`form:"Amount1"`
	Amount 		sql.NullFloat64 	`form:"Amount"`
}

// Relations
type KitchenContains struct {
	Item   	string 	`form:"Item"`
	Amount 	sql.NullFloat64 		`form:"Amount"`
	Unit	string 
}



func main() {
	m := martini.Classic()
	// render html templates from templates directory
	m.Use(render.Renderer(render.Options{
		Extensions: []string{".tmpl", ".html"},
	}))

	m.Use(dbHandler())

	m.Get("/", func(r render.Render, db *gorp.DbMap) {
		var recipes []Recipe // Where to save the DB SELECT
		_, _ = db.Select(&recipes, "SELECT * FROM recipe")
		var kitchens []KitchenContains // Where to save the DB SELECT
		_, _ = db.Select(&kitchens, "SELECT item, COALESCE(amount, -1) AS amount, unit FROM kitchen LEFT JOIN food ON food.name=kitchen.item")

		// Can do

		var recipe_can []Recipe // Where to save the DB SELECT
		_, _ = db.Select(&recipe_can, "SELECT recipe.name FROM recipe WHERE recipe.name NOT IN (SELECT DISTINCT recipe_ingredients.name FROM recipe_ingredients LEFT JOIN kitchen ON kitchen.item=recipe_ingredients.foodname WHERE (kitchen.amount < recipe_ingredients.amount OR kitchen.item IS NULL) OR kitchen.amount IS NULL)")
		// Maybe can do
		var recipe_maybe []Recipe // Where to save the DB SELECT
		_, _ = db.Select(&recipe_maybe, "SELECT recipe.name FROM recipe WHERE recipe.name NOT IN (SELECT DISTINCT recipe_ingredients.name FROM recipe_ingredients LEFT JOIN kitchen ON kitchen.item=recipe_ingredients.foodname WHERE (kitchen.amount < recipe_ingredients.amount OR kitchen.item IS NULL)) EXCEPT SELECT recipe.name FROM recipe WHERE recipe.name NOT IN (SELECT DISTINCT recipe_ingredients.name FROM recipe_ingredients LEFT JOIN kitchen ON kitchen.item=recipe_ingredients.foodname WHERE (kitchen.amount < recipe_ingredients.amount OR kitchen.item IS NULL) OR kitchen.amount IS NULL)")
		// Cannot do
		var recipe_cant []Recipe // Where to save the DB SELECT
		_, _ = db.Select(&recipe_cant, "SELECT recipe.name FROM recipe WHERE recipe.name IN (SELECT DISTINCT recipe_ingredients.name FROM recipe_ingredients LEFT JOIN kitchen ON kitchen.item=recipe_ingredients.foodname WHERE (kitchen.amount < recipe_ingredients.amount OR kitchen.item IS NULL)) ")

		for index, recipe := range recipes {
			recipes[index].Possible = "no"
			fmt.Println(recipe)
			for _, can := range recipe_can {
				if recipe.Name == can.Name {
					recipes[index].Possible = "success"
				} 
			}
			for _, maybe := range recipe_maybe {
				if recipe.Name == maybe.Name {
					recipes[index].Possible = "warning"
				} 
			}
			for _, cant := range recipe_cant {
				if recipe.Name == cant.Name {
					recipes[index].Possible = "danger"
				} 
			}
		}

		data := map[string]interface{}{"title": "Receptdatabas", "recipes": recipes, "kitchen": kitchens}
		
		// Response code, title of template, input for template
		r.HTML(200, "index", data)
	})

	m.Get("/items", func(r render.Render, db *gorp.DbMap) {
		var kitchens []Kitchen // Where to save the DB SELECT
		_, _ = db.Select(&kitchens, "SELECT * FROM kitchen") // Query
		data := map[string]interface{}{"title": "Hej", "body": "Här händer det mycket", "kitchen": kitchens}
		r.HTML(200, "list", data)
	})

	// binding.Form = magic to bind a struct to elements from a form
	m.Post("/kitchen", binding.Form(KitchenForm{}), func(kitchen KitchenForm, r render.Render, db *gorp.DbMap) {
		fmt.Println(kitchen)
		var newKitchen Kitchen
		newKitchen.Item = kitchen.Item
		newKitchen.Amount.Float64 = kitchen.Amount
		if kitchen.Amount == 0.0 {
			//kitchen.Amount = nil
			newKitchen.Amount.Valid = false
		} else {
			newKitchen.Amount.Valid = true
		}
		err := db.SelectOne(&newKitchen, "SELECT * FROM kitchen WHERE Item = $1", newKitchen.Item)

		// If item exists
		if err == nil {
			fmt.Println("Trying to update kitchen")
			_, err = db.Update(&newKitchen)
		} else {
			fmt.Println("Trying to insert into kitchen")
			// Check if food type exists
			var food Food
			err := db.SelectOne(&food, "SELECT * FROM food WHERE name = $1", newKitchen.Item)
			fmt.Println(err)
			if err == nil {
				food.Name = newKitchen.Item
				food.Unit = kitchen.Unit
				err = db.Insert(&food)
				fmt.Println(food)
				fmt.Println(err)
			}
			_, err = db.Exec("INSERT INTO kitchen (Item, Amount) VALUES ($1, $2)", newKitchen.Item, newKitchen.Amount)
			//err = db.Insert(&newKitchen)
			fmt.Println(newKitchen)
			fmt.Println(err)
		}

		r.Redirect("/", 301)
	})
	m.Run()
}

// Database
func initDb() *gorp.DbMap {
	// connect to db using standard Go database/sql API
	// use whatever database/sql driver you wish
	db, err := sql.Open("postgres", "user=joppe dbname=lab2 sslmode=disable")
	checkErr(err, "sql.Open failed")

	// construct a gorp DbMap
	dbmap := &gorp.DbMap{Db: db, Dialect: gorp.SqliteDialect{}}

	// add a table, setting the table name to 'posts' and
	// specifying that the Id property is an auto incrementing PK
	dbmap.AddTableWithName(Kitchen{}, "kitchen").SetKeys(false, "Item")
	dbmap.AddTableWithName(Food{}, "food").SetKeys(false, "Name")
	dbmap.AddTableWithName(Recipe{}, "recipe").SetKeys(false, "Name")
	dbmap.AddTableWithName(RecipeIngredients{}, "recipe_ingredients").SetKeys(false, "Name").SetKeys(false, "FoodName")

	// create the table. in a production system you'd generally
	// use a migration tool, or create the tables via scripts
	err = dbmap.CreateTablesIfNotExists()
	checkErr(err, "Create tables failed")

	return dbmap
}

func checkErr(err error, msg string) {
	if err != nil {
		fmt.Println(err)
		fmt.Println(msg)
	}
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
