// main.go
package main

import (
	"database/sql"
	"fmt"
	"github.com/coopernurse/gorp"
	"github.com/go-martini/martini"
	_ "github.com/lib/pq"
	"github.com/martini-contrib/binding"
	"github.com/martini-contrib/render"
	_ "github.com/mattn/go-sqlite3"
	"net/http"
	_ "strconv"
	"strings"
)

// Database
type Kitchen struct {
	Item   string
	Amount sql.NullFloat64
}

type KitchenForm struct {
	Item    string  `form:"Item"`
	Amount  float64 `form:"Amount"`
	Unit    string  `form:"Unit"`
	Unknown string  `form:"Unknown"`
}

type Food struct {
	Name string
	Unit string
}

type Recipe struct {
	Name        string `form:"Item"`
	Type        string `form:"Type"`
	Description string `form:"Description"`
	Possible    string
}

type RecipeIngredients struct {
	Name     string
	FoodName string
	Amount   sql.NullFloat64
}

type RecipeMake struct {
	Name          string
	FoodName      string
	Unit          string
	RecipeAmount  sql.NullFloat64
	KitchenAmount sql.NullFloat64
	Difference    sql.NullFloat64
	AbsDifference sql.NullFloat64
}

// Relations
type KitchenContains struct {
	Item   string
	Amount sql.NullFloat64
	Unit   string
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

		var recipe_can []Recipe 
		var recipe_maybe []Recipe
		var recipe_cant []Recipe
		_, _ = db.Select(&recipe_can, "SELECT recipe.name FROM recipe " +
										"WHERE recipe.name NOT IN ("+
											"SELECT DISTINCT recipe_ingredients.name "+
											"FROM recipe_ingredients "+
											"LEFT JOIN kitchen ON kitchen.item=recipe_ingredients.foodname "+
											"WHERE (kitchen.amount < recipe_ingredients.amount OR kitchen.item IS NULL) "+
											"OR kitchen.amount IS NULL "+
										")")
		_, _ = db.Select(&recipe_maybe, "SELECT recipe.name FROM recipe "+
										"WHERE recipe.name NOT IN ("+
											"SELECT DISTINCT recipe_ingredients.name "+
											"FROM recipe_ingredients "+
											"LEFT JOIN kitchen ON kitchen.item=recipe_ingredients.foodname "+
											"WHERE (kitchen.amount < recipe_ingredients.amount OR kitchen.item IS NULL)"+
										") "+
										"EXCEPT "+ 
										"SELECT recipe.name FROM recipe "+
										"WHERE recipe.name NOT IN ("+
											"SELECT DISTINCT recipe_ingredients.name FROM recipe_ingredients "+
											"LEFT JOIN kitchen ON kitchen.item=recipe_ingredients.foodname "+
											"WHERE (kitchen.amount < recipe_ingredients.amount OR kitchen.item IS NULL) "+
											"OR kitchen.amount IS NULL
										)")

		_, _ = db.Select(&recipe_cant, "SELECT recipe.name FROM recipe "+
										"WHERE recipe.name IN ("+
											"SELECT DISTINCT recipe_ingredients.name FROM recipe_ingredients "+
											"LEFT JOIN kitchen ON kitchen.item=recipe_ingredients.foodname "+
											"WHERE (kitchen.amount < recipe_ingredients.amount OR kitchen.item IS NULL)"+
										") ")

		// Link to all of them
		var recipe_all string
		for index, recipe := range recipes {
			recipes[index].Possible = "no"
			recipe_all = recipe_all + "recipe=" + recipe.Name + "&"
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
		recipe := "'" + strings.Join(recipes, "' OR ri.name='") + "'"
		recipeNames := strings.Join(recipes, ", ")
		// Select all ingredients necessary for making each dish

		// I try make stuff -Nisse
		//Ingredients_needed for the making of chosen recipes
		var ingredients_needed []RecipeMake
		_, err := db.Select(&ingredients_needed, "SELECT ri.foodname AS FoodName, "+
			"SUM(ri.amount) AS RecipeAmount, food.unit AS Unit FROM recipe_ingredients AS ri "+
			"LEFT JOIN food ON food.name = ri.foodname "+
			"WHERE (ri.name ="+recipe+") GROUP BY ri.foodname, food.unit")
		fmt.Println("ingredients_needed, err: ", err)
		//What is in the kitchen
		fmt.Println("ingredients_needed: ", ingredients_needed)
		var kitchen_contains []Kitchen
		_, err = db.Select(&kitchen_contains, "SELECT * FROM kitchen")
		fmt.Println("kitchen_contains: ", kitchen_contains)

		//var ingredients_use_temp []RecipeMake

		var ingredients_use []RecipeMake
		var ingredients_maybe []RecipeMake
		var ingredients_not []RecipeMake
		var ingredients_required []RecipeMake
		//test

		//Lyckas sortera ingredienser över huruvida de finns i köket!
		for _, v := range ingredients_needed {
			fmt.Println("Iterating item: ", v.FoodName)
			if strInKitchen(v.FoodName, kitchen_contains) {
				//Elementet finns representerat i köket
				fmt.Println(v.FoodName, " Was represented in the kitchen.")
				amount_kitchen := amountInKitchen(v.FoodName, kitchen_contains)
				if amount_kitchen.Valid == true && amount_kitchen.Float64 > 0 {
					//Valid värde i köket.
					fmt.Println(v.FoodName, " Has a valid amount.")
					//Utför en kontroll hur värdet förhåller sig med receptet.
					if amount_kitchen.Float64 >= v.RecipeAmount.Float64 {
						//Ingrediensen finns i köket, amount valid, tillräcklig mängd.
						//Lägg till elemetet i ingredients_use
						v.Difference = sql.NullFloat64{Float64: (amount_kitchen.Float64 - v.RecipeAmount.Float64), Valid: true}
						ingredients_use = append(ingredients_use, v)
					} else { //Otillräcklig mängd i köket, behöver handla
						//Lägg in i ingredients_not
						ingredients_not = append(ingredients_not, v)
					}
				} else { //Null alternativt olämpligt värde köket.
					fmt.Println(v.FoodName, " Has an unvalid amount.")
					//Lägg till elementet i ingredients_maybe

					ingredients_maybe = append(ingredients_maybe, v)
				}
			} else { //Elementet finns inte i köket över huvud. Kontroll om den finns i food?
				fmt.Println(v.FoodName, " is not in the kitchen.")
				//Lägg till elementet i ingredients_not
				ingredients_not = append(ingredients_not, v)
			}
		}

		ingredients := map[string]interface{}{"recipe": recipeNames, "can": ingredients_use, "maybe": ingredients_maybe, "cannot": ingredients_not, "required": ingredients_required}
		fmt.Println("ingredients: ", ingredients)
		data := map[string]interface{}{"title": "Make a dish", "recipe": recipes, "create_recipes": ingredients}
		r.HTML(200, "make", data)
	})

	m.Get("/created", func(r render.Render, req *http.Request, db *gorp.DbMap) {
		all_recipes := req.URL.Query().Get("recipe")
		recipes := strings.Split(all_recipes, ", ")
		recipe := "'" + strings.Join(recipes, "' OR ri.name='") + "'"

		var ingredients []RecipeMake
		_, err := db.Select(&ingredients, "SELECT ri.name AS Name, ri.foodname AS FoodName, food.unit, ri.amount AS RecipeAmount, kitchen.amount AS KitchenAmount, (kitchen.amount - ri.amount) AS Difference, COALESCE(ABS(kitchen.amount - ri.amount), ri.amount) AS AbsDifference"+
			" FROM recipe_ingredients AS ri LEFT JOIN kitchen ON kitchen.item=ri.foodname "+
			"LEFT JOIN food ON food.name=ri.foodname WHERE (ri.name="+recipe+
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

		r.Redirect("/", 302)
	})

	// binding.Form = magic to bind a struct to elements from a form
	m.Post("/kitchen", binding.Form(KitchenForm{}), func(kitchen KitchenForm, r render.Render, db *gorp.DbMap) {
		var newKitchen Kitchen
		err := db.SelectOne(&newKitchen, "SELECT * FROM kitchen WHERE Item = $1", kitchen.Item)

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
			var food Food
			err := db.SelectOne(&food, "SELECT * FROM food WHERE name = $1", newKitchen.Item)

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
			}

			err = db.Insert(&newKitchen)
			checkErr(err, "Could not insert new kitchen item")
		}

		r.Redirect("/", 302)
	})

	m.Post("/kitchen_remove", binding.Form(KitchenForm{}), func(kitchen KitchenForm, r render.Render, db *gorp.DbMap) {
		fmt.Println(kitchen)
		var newKitchen Kitchen
		err := db.SelectOne(&newKitchen, "SELECT * FROM kitchen WHERE kitchen.Item = $1", kitchen.Item)

		// Update new kitchen's amount from what is already in db
		// If item exists
		if err == nil {
			//Not null in database
			if newKitchen.Amount.Valid == true { 
				newKitchen.Amount.Float64 = newKitchen.Amount.Float64 - kitchen.Amount
				if newKitchen.Amount.Float64 > 0 && kitchen.Unknown != "true" { //Still positive, and user do not want to delete.
					_, err = db.Update(&newKitchen)

				// Not positive, alternatively, it is set to null.
				} else {
					fmt.Println("")
					_, err = db.Delete(&newKitchen)
				}
			// Null in the database (previous value) && the user wants to remove it.
			} else if kitchen.Unknown == "true" { 
				_, err = db.Delete(&newKitchen)
			} 
		}

		r.Redirect("/", 301)
	})

	m.Run()
}

// Database
func initDb() *gorp.DbMap {
	// Connect to db using standard Go database/sql API
	db, err := sql.Open("postgres", "user=nisse password=nisse dbname=lab2 sslmode=disable")
	checkErr(err, "sql.Open failed")

	// Construct a gorp DbMap
	dbmap := &gorp.DbMap{Db: db, Dialect: gorp.PostgresDialect{}}

	// add all tables, first argument of SetKeys is auto incrementing t/f
	dbmap.AddTableWithName(Kitchen{}, "kitchen").SetKeys(false, "Item")
	dbmap.AddTableWithName(Food{}, "food").SetKeys(false, "Name")
	dbmap.AddTableWithName(Recipe{}, "recipe").SetKeys(false, "Name")
	dbmap.AddTableWithName(RecipeIngredients{}, "recipe_ingredients").SetKeys(false, "Name").SetKeys(false, "FoodName")

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

//Control if an item (string) is represented in a Kitchen-struct.
//Makes the assumption that if the item is in the Kitchen, then it is in the database Food.
//Don't kow what that might lead to.
func strInKitchen(str string, kitchen []Kitchen) bool {
	for _, b := range kitchen {
		if str == b.Item {
			return true
		}
	}
	return false
}

// Same as strInKitchen but returns the amount for the item
func amountInKitchen(item string, kitchen []Kitchen) sql.NullFloat64 {
	for _, b := range kitchen {
		if item == b.Item {
			return b.Amount
		}
	}
	return sql.NullFloat64{Float64: 0, Valid: false}
}

// Middleware for database connection
func dbHandler() martini.Handler {
	// Return a martini.Handler to be called for every request
	return func(c martini.Context) {
		dbmap := initDb()
		c.Map(dbmap)
		defer dbmap.Db.Close()
		c.Next()
	}
}
