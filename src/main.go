package main

import (
	"fmt"
	"time"

	recipe "github.com/Henshou/Tubes2_BE_CraftingTable.git/recipe"
	scraper "github.com/Henshou/Tubes2_BE_CraftingTable.git/scraper"
)

func main() {
	fmt.Println("Starting Little Alchemy 2 recipe finder...")
	start := time.Now()

	// Get recipes for every element
	scraper.FindRecipes()
	var err error
	recipe.RecipeMap, err = recipe.ReadJson("recipes.json")
	if err != nil {
		fmt.Println("Error reading JSON file:", err)
		return
	}
	name := recipe.RecipeMap["Brick"].Name
	fmt.Println("Name:", name)
	bus := &recipe.RecipeTreeNode{Name: name}
	var ValidRecipes []string

	str, _ := recipe.BuildRecipeTreeBFS(bus, 30)

	fmt.Println("Recipe tree string:", str)
	// recipe.PrintRecipeTree(bus, "")
	fmt.Println("Recipe tree built successfully.")
	// fmt.Println(recipe.IsBaseElement("Fire"))
	fmt.Println("Recipe tree string:", ValidRecipes)
	end := time.Now()
	fmt.Printf("Execution time: %v\n", end.Sub(start))
	// fmt.Printf("number of recipes: %d\n", num)
	// fmt.Println("Recipe tree printed successfully.")
}

//how to run this code?
// go run src/main.go
//how to compile this code?
// go build -o main src/main.go
//how to run without go build
// go run src/main.go
