package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	recipe "github.com/Henshou/Tubes2_BE_CraftingTable.git/recipe"
)

func Start() {
	var err error
	recipe.RecipeMap, err = recipe.ReadJson("recipes.json")
	if err != nil {
		log.Fatalf("Failed to read recipe JSON: %v", err)
	}

	http.HandleFunc("/api/concurrentDFS", concurrentDFSHandler)

	fmt.Println("Server started on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))

}

func concurrentDFSHandler(w http.ResponseWriter, r *http.Request) {
	target := r.URL.Query().Get("target")
	if target == "" {
		http.Error(w, "Missing 'target' query param", http.StatusBadRequest)
		return
	}

	countStr := r.URL.Query().Get("count")
	count := 0
	if countStr != "" {
		var err error
		count, err = strconv.Atoi(countStr)
		if err != nil {
			http.Error(w, "Invalid 'count' query param (must be an integer)", http.StatusBadRequest)
			return
		}
	}

	root := recipe.BuildTreeWithLimit(target, recipe.RecipeMap, count)
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(root); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
