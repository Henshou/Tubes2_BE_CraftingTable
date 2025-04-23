package main

import (
	"fmt"

	scrapper "github.com/Henshou/Tubes2_BE_CraftingTable.git/scraper"
)

func main() {
	fmt.Println("Starting Little Alchemy 2 recipe finder...")

	// Choose the element you want to find recipes for
	targetElement := "Dust"

	// Get recipes for the target element
	recipes := scrapper.FindRecipes(targetElement)

	if len(recipes) == 0 {
		fmt.Printf("No recipes found for '%s'\n", targetElement)
		return
	}

	fmt.Printf("Found %d recipes for '%s':\n", len(recipes), targetElement)
	for i, recipe := range recipes {
		fmt.Printf("  %d. %s + %s\n", i+1, recipe[0], recipe[1])
	}
}
