// main.go
package main

import (
	"database/sql"
	"github.com/go-martini/martini"
	"github.com/coopernurse/gorp"
	"github.com/martini-contrib/binding"
	"github.com/martini-contrib/render"
	"net/http"
	_"github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	_"strconv"
	"fmt"
	"strings"
)

// Database
type Kitchen struct {
	Item   string
	Amount sql.NullFloat64
}

type KitchenForm struct {
	Item   string 		`form:"Item"`
	Amount float64 		`form:"Amount"`
	Unit string 		`form:"Unit"`
	Unknown string 		`form:"Unknown"`
}

type Food struct {
	Name string
	Unit string
}

type Recipe struct {
	Name 		string 	`form:"Item"`
	Type 		string 	`form:"Type"`
	Description string 	`form:"Description"`
	Possible	string
}

type RecipeIngredients struct {
	Name 		string
	FoodName 	string
	Amount 		sql.NullFloat64
}

type RecipeMake struct {
	Name 		string
	FoodName 	string
	Unit 		string
	RecipeAmount sql.NullFloat64
	KitchenAmount sql.NullFloat64
	Difference sql.NullFloat64
	AbsDifference sql.NullFloat64

}

// Relations
type KitchenContains struct {
	Item   	string
	Amount 	sql.NullFloat64
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

		// Link to all of them
		var recipe_all string
		for index, recipe := range recipes {
			recipes[index].Possible = "no"
			recipe_all = recipe_all + "recipe="+recipe.Name+"&"
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

		data := map[string]interface{}{"title": "Receptdatabas", "recipes": recipes, "all_recipes": recipe_all, "kitchen": kitchens}
		
		// Response code, title of template, input for template
		r.HTML(200, "index", data)
	})

	m.Get("/make", func(r render.Render, req *http.Request, db *gorp.DbMap) {
		recipes := req.URL.Query()["recipe"]


		//create_recipes := make([]map[string]interface{}, len(recipes))
//		for i, recipe := range recipes {
		recipe := "'" + strings.Join(recipes, "' OR ri.name='") + "'"
		fmt.Println(recipe)
		recipeNames := strings.Join(recipes,", ")
			// Select all ingredients necessary for making each dish
			// Get all ingredients we definitely can use
			var ingredients_use []RecipeMake 
			_, err := db.Select(&ingredients_use, "SELECT ri.name AS Name, ri.foodname AS FoodName, food.unit, ri.amount AS RecipeAmount, kitchen.amount AS KitchenAmount, (kitchen.amount - ri.amount) AS Difference, COALESCE(ABS(kitchen.amount - ri.amount), ri.amount) AS AbsDifference" +
				" FROM recipe_ingredients AS ri LEFT JOIN kitchen ON kitchen.item=ri.foodname LEFT JOIN food ON food.name=ri.foodname WHERE (ri.name=" + recipe +
				") AND (kitchen.amount - ri.amount) >= 0")
			checkErr(err, "Selecting ingredients we can use")

			// Get all ingredients we maybe can use
			var ingredients_maybe []RecipeMake 
			_, err = db.Select(&ingredients_maybe, "SELECT ri.name AS Name, ri.foodname AS FoodName, food.unit, ri.amount AS RecipeAmount, kitchen.amount AS KitchenAmount, (kitchen.amount - ri.amount) AS Difference, COALESCE(ABS(kitchen.amount - ri.amount), ri.amount) AS AbsDifference" +
				" FROM recipe_ingredients AS ri LEFT JOIN kitchen ON kitchen.item=ri.foodname LEFT JOIN food ON food.name=ri.foodname WHERE (ri.name=" + recipe +
				") AND kitchen.amount IS NULL")
			checkErr(err, "Selecting ingredients we maybe can use")

			// Get all ingredients we cannot use
			var ingredients_not []RecipeMake 
			_, err = db.Select(&ingredients_not, "SELECT ri.name AS Name, ri.foodname AS FoodName, food.unit, ri.amount AS RecipeAmount, kitchen.amount AS KitchenAmount, (kitchen.amount - ri.amount) AS Difference, COALESCE(ABS(kitchen.amount - ri.amount), ri.amount) AS AbsDifference" +
				" FROM recipe_ingredients AS ri LEFT JOIN kitchen ON kitchen.item=ri.foodname LEFT JOIN food ON food.name=ri.foodname WHERE (ri.name=" + recipe +
				") AND ((kitchen.amount - ri.amount) < 0 OR kitchen.item IS NULL)")
			checkErr(err, "Selecting ingredients we cannot use")


			ingredients := map[string]interface{}{"recipe": recipeNames, "can": ingredients_use, "maybe": ingredients_maybe, "cannot": ingredients_not}
			//create_recipes[i] = ingredients
//		}
		fmt.Println(ingredients)
		data := map[string]interface{}{"title": "Make a dish", "recipe": recipes, "create_recipes": ingredients}
		r.HTML(200, "make", data)
	})

	m.Get("/created", func(r render.Render, req *http.Request, db *gorp.DbMap) {
		all_recipes := req.URL.Query().Get("recipe")
		recipes := strings.Split(all_recipes, ", ")
		recipe := "'" + strings.Join(recipes, "' OR ri.name='") + "'"

		var ingredients []RecipeMake 
		_, err := db.Select(&ingredients, "SELECT ri.name AS Name, ri.foodname AS FoodName, food.unit, ri.amount AS RecipeAmount, kitchen.amount AS KitchenAmount, (kitchen.amount - ri.amount) AS Difference, COALESCE(ABS(kitchen.amount - ri.amount), ri.amount) AS AbsDifference" +
				" FROM recipe_ingredients AS ri LEFT JOIN kitchen ON kitchen.item=ri.foodname " +
				"LEFT JOIN food ON food.name=ri.foodname WHERE (ri.name=" + recipe +
				")")
		checkErr(err, "Selecting ingredients to remove from kitchen")

		for _, ingredient := range ingredients {
			// Nothing left in kitchen
			if ingredient.Difference.Valid == false {
				// Well? 
			} else if ingredient.Difference.Float64 <= 0 {
				db.Exec("DELETE FROM kitchen WHERE Item=$1", ingredient.FoodName)
			} else {
				var kitchenItem Kitchen
				err = db.SelectOne(&kitchenItem, "SELECT * FROM kitchen WHERE Item=$1", ingredient.FoodName)
				checkErr(err, "Select item from kitchen from ingredient")
				kitchenItem.Amount = ingredient.Difference
				_, err = db.Update(&kitchenItem)
				checkErr(err, "Update amount in kitchen")
			}
		}

		r.Redirect("/", 300)
	})

	// binding.Form = magic to bind a struct to elements from a form
	m.Post("/kitchen", binding.Form(KitchenForm{}), func(kitchen KitchenForm, r render.Render, db *gorp.DbMap) {
		fmt.Println(kitchen)
		var newKitchen Kitchen
		err := db.SelectOne(&newKitchen, "SELECT * FROM kitchen WHERE Item = $1", kitchen.Item)
		
		// Update new kitchen's amount from what is already in db
		// If item exists
		if err == nil {
			fmt.Println("Trying to update kitchen")
			newKitchen.Amount.Float64 = newKitchen.Amount.Float64 + kitchen.Amount
			if (kitchen.Amount == 0.0 || newKitchen.Amount.Valid == false) && kitchen.Unknown != "true" {
				newKitchen.Amount.Valid = false
			} else {
				newKitchen.Amount.Valid = true
			}

			_, err = db.Update(&newKitchen)
			checkErr(err, "Updating kitchen item")
		} else {
			fmt.Println("Trying to insert into kitchen")
			// Check if food type exists
			var food Food
			err := db.SelectOne(&food, "SELECT * FROM food WHERE name = $1", newKitchen.Item)
			fmt.Println(err)


			newKitchen.Item = kitchen.Item
			newKitchen.Amount.Float64 = kitchen.Amount
			newKitchen.Amount.Valid = true
			if kitchen.Amount == 0.0 {
				newKitchen.Amount.Valid = false
			}

			if err != nil {
				food.Name = newKitchen.Item
				food.Unit = kitchen.Unit
				err = db.Insert(&food)
				fmt.Println(food)
				fmt.Println(err)
			}

			err = db.Insert(&newKitchen)
			checkErr(err, "Inserting new kitchen item")
		}

		r.Redirect("/", 300)
	})
	m.Run()
}

// Database
func initDb() *gorp.DbMap {
	// connect to db using standard Go database/sql API
	// use whatever database/sql driver you wish
	db, err := sql.Open("postgres", "user=nisse password=nisse dbname=lab2 sslmode=disable")
	checkErr(err, "sql.Open failed")

	// construct a gorp DbMap
	dbmap := &gorp.DbMap{Db: db, Dialect: gorp.PostgresDialect{}}

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
