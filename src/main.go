// main.go
package main

import (
	"log"
	"sync"

	"github.com/Henshou/Tubes2_BE_CraftingTable.git/recipe"
	"github.com/Henshou/Tubes2_BE_CraftingTable.git/scraper"
)

func main() {
	// 1) Scrape all recipes into recipes.json
	log.Println("Scraping recipes...")
	scraper.FindRecipes()
	log.Println("Done scraping. Written recipes.json.")

	// 2) Load recipes.json into RecipeMap
	var err error
	recipe.RecipeMap, err = recipe.ReadJson("recipes.json")
	if err != nil {
		log.Fatalf("Failed to load recipes.json: %v", err)
	}
	log.Printf("Loaded %d recipes.\n", len(recipe.RecipeMap))

	bus := &recipe.RecipeTreeNode{Name: "Life"}
	stopChan := make(chan bool)
	wg := &sync.WaitGroup{}
	mu := &sync.Mutex{}

	// Start the receiver goroutine to listen for stop signal
	go recipe.StopSearch(stopChan, wg)

	// Start the recipe tree generation concurrently
	wg.Add(1)
	go recipe.BuildRecipeTreeBFS(bus, recipe.RecipeMap, 43, stopChan, wg, mu)
	wg.Wait()

	log.Println(recipe.CalculateTotalCompleteRecipes(bus))
	recipe.PruneTree(bus)
	recipe.PrintRecipeTree(bus, "")
	// 3) Start HTTP API server

	// server.Start()
}

/*
package main

import (
	"fmt"
	"time"

	recipe "github.com/Henshou/Tubes2_BE_CraftingTable.git/recipe"
	scraper "github.com/Henshou/Tubes2_BE_CraftingTable.git/scraper"
	server "github.com/Henshou/Tubes2_BE_CraftingTable.git/server"
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

	recipe.BuildRecipeTreeBFSConcurrent(bus, recipe.RecipeMap, 5, &ValidRecipes)

	fmt.Println("Recipe tree string:", ValidRecipes)
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
// go run src/main.go */
