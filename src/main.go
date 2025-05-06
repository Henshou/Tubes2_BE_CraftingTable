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
	name := recipe.RecipeMap["Fire"].Name
	fmt.Println("Name:", name)
}
