package main

import (
	"fmt"
	scraper "github.com/Henshou/Tubes2_BE_CraftingTable.git/scraper"
)

func main() {
	fmt.Println("Starting Little Alchemy 2 recipe finder...")

	// Get recipes for every element
	scraper.FindRecipes()

}
