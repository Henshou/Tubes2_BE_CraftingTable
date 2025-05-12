package main

import (
	"log"

	"github.com/Henshou/Tubes2_BE_CraftingTable.git/recipe"
	"github.com/Henshou/Tubes2_BE_CraftingTable.git/scraper"
	"github.com/Henshou/Tubes2_BE_CraftingTable.git/server"
)

func main() {
	log.Println("Scraping recipes…")
	scraper.FindRecipes()
	log.Println("Finished scraping; wrote recipes.json")

	var err error
	recipe.RecipeMap, err = recipe.ReadJson("recipes.json")
	if err != nil {
		log.Fatalf("Failed to load recipes.json: %v", err)
	}
	log.Printf("Loaded %d recipes.\n", len(recipe.RecipeMap))

	log.Println("Starting HTTP server on :8080")
	server.Start()
}
