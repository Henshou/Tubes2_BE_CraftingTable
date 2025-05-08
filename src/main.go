package main

import (
	"fmt"
	"sync"
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
	name := recipe.RecipeMap["Lava"].Name
	fmt.Println("Name:", name)
	bus := &recipe.RecipeTreeNode{Name: name}

	wg := &sync.WaitGroup{}
	mu := &sync.Mutex{}
	wg.Add(1)
	go recipe.BuildRecipeTreeDFSConcurrent(bus, recipe.RecipeMap, wg, mu)
	wg.Wait()
	fmt.Println("Recipe tree built successfully.")
	recipe.PrintRecipeTree(bus, "")
	end := time.Now()
	fmt.Printf("Execution time: %v\n", end.Sub(start))
	test := recipe.MaxQueueLength
	fmt.Println("Max queue length:", test)
	// fmt.Println("Recipe tree printed successfully.")
}

//how to run this code?
// go run src/main.go
//how to compile this code?
// go build -o main src/main.go
//how to run without go build
// go run src/main.go
