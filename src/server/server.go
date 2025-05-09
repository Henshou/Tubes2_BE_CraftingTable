// server.go
package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"

	recipe "github.com/Henshou/Tubes2_BE_CraftingTable.git/recipe"
)

func Start() {
	// load recipes.json once at startup
	var err error
	recipe.RecipeMap, err = recipe.ReadJson("recipes.json")
	if err != nil {
		log.Fatalf("Failed to load recipes.json: %v", err)
	}

	http.HandleFunc("/api/recipes", recipesHandler)
	http.HandleFunc("/api/dfs",      dfsHandler)
	http.HandleFunc("/api/bfs",      bfsHandler)
	http.HandleFunc("/api/bidirectional", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Bidirectional search not implemented", http.StatusNotImplemented)
	})

	fmt.Println("Server listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// recipesHandler just serves the raw recipes.json
func recipesHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	data, err := os.ReadFile("recipes.json")
	if err != nil {
		http.Error(w, "cannot read recipes.json", http.StatusInternalServerError)
		return
	}
	w.Write(data)
}

// dfsHandler builds a tree via concurrent DFS, up to `count` full recipes
func dfsHandler(w http.ResponseWriter, r *http.Request) {
	target := r.URL.Query().Get("target")
	if target == "" {
		http.Error(w, "missing target", http.StatusBadRequest)
		return
	}
	count := parseCount(r)

	// reset global state
	recipe.VisitedMap = make(map[string]*recipe.RecipeTreeNode)

	// collect up to count recipes (for testing/demo; not used by frontend)
	validRecipes := []string{}

	// build
	root := &recipe.RecipeTreeNode{Name: target}
	wg := &sync.WaitGroup{}
	mu := &sync.Mutex{}
	wg.Add(1)
	go recipe.BuildRecipeTreeDFSConcurrent(root, recipe.RecipeMap, wg, mu, count, &validRecipes)
	wg.Wait()

	writeJSON(w, root)
}

// bfsHandler builds a tree via concurrent BFS, up to `count` full recipes
func bfsHandler(w http.ResponseWriter, r *http.Request) {
	target := r.URL.Query().Get("target")
	if target == "" {
		http.Error(w, "missing target", http.StatusBadRequest)
		return
	}
	count := parseCount(r)

	// reset
	recipe.VisitedMap = make(map[string]*recipe.RecipeTreeNode)

	validRecipes := []string{}

	// build
	root := &recipe.RecipeTreeNode{Name: target}
	recipe.BuildRecipeTreeBFSConcurrent(root, recipe.RecipeMap, count, &validRecipes)

	writeJSON(w, root)
}

// parseCount reads ?count= or defaults to 1
func parseCount(r *http.Request) int {
	if s := r.URL.Query().Get("count"); s != "" {
		if c, err := strconv.Atoi(s); err == nil {
			return c
		}
	}
	return 1
}

// writeJSON is a small helper to marshal & write
func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		http.Error(w, "json encode error", http.StatusInternalServerError)
	}
}
