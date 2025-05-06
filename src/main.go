package main

import (
	"fmt"

	recipe "github.com/Henshou/Tubes2_BE_CraftingTable.git/recipe"
	scraper "github.com/Henshou/Tubes2_BE_CraftingTable.git/scraper"
)

func main() {
	fmt.Println("Starting Little Alchemy 2 recipe finder...")

	// Get recipes for every element
	scraper.FindRecipes()
	var err error
	recipe.RecipeMap, err = recipe.ReadJson("recipes.json")
	if err != nil {
		fmt.Println("Error reading JSON file:", err)
		return
	}
	name := recipe.RecipeMap["Excalibur"].Name
	fmt.Println("Name:", name)
	bus := &recipe.RecipeTreeNode{Name: name}
	recipe.BuildRecipeTreeBFS(bus)
	fmt.Println("Recipe tree built successfully.")
	recipe.PrintRecipeTree(bus, "", true)
	fmt.Println("Recipe tree printed successfully.")
	// recipe.BFSPrintRecipeTree(bus)
	// fmt.Println("BFS recipe tree printed successfully.")
}

//how to run this code?
// go run src/main.go
//how to compile this code?
// go build -o main src/main.go
