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
	"time"

	recipe "github.com/Henshou/Tubes2_BE_CraftingTable.git/recipe"
)

// NodeDTO is exactly what the front-end’s Tree.jsx expects.
type NodeDTO struct {
	Name    string      `json:"name"`
	Recipes []RecipeDTO `json:"recipes"`
}

type RecipeDTO struct {
	Inputs []NodeDTO `json:"inputs"`
}

// TreeResponse wraps the full tree plus metrics.
type TreeResponse struct {
	Tree         NodeDTO `json:"tree"`
	TimeTaken    int64   `json:"timeTaken"`    // ms
	NodesVisited int     `json:"nodesVisited"`
	RecipesFound int     `json:"recipesFound"`
	MethodUsed   string  `json:"methodUsed"`
}

// buildDTO walks your in-memory RecipeTreeNode and emits a NodeDTO.
func buildDTO(node *recipe.RecipeTreeNode) NodeDTO {
	dto := NodeDTO{
		Name:    node.Name,
		Recipes: make([]RecipeDTO, 0),
	}
	for _, group := range node.Children {
		if len(group) == 0 {
			continue
		}
		inputs := make([]NodeDTO, 0, len(group))
		for _, child := range group {
			inputs = append(inputs, buildDTO(child))
		}
		dto.Recipes = append(dto.Recipes, RecipeDTO{Inputs: inputs})
	}
	return dto
}

// writeJSON is a helper for one-shot JSON responses.
func writeJSON(w http.ResponseWriter, v interface{}) {
  w.Header().Set("Access-Control-Allow-Origin", "*")
  w.Header().Set("Content-Type", "application/json")

  // marshal into a []byte so we can both log it and send it
  payload, err := json.Marshal(v)
  if err != nil {
    log.Printf("[writeJSON] ✗ marshal error: %v\n", err)
    http.Error(w, "json encode error", http.StatusInternalServerError)
    return
  }
  
  // debug log the exact JSON we're about to send
  log.Printf("[writeJSON] → sending %d bytes: %s\n", len(payload), payload)

  // write it out
  if _, err := w.Write(payload); err != nil {
    log.Printf("[writeJSON] ✗ write error: %v\n", err)
  }
}

// parseCount reads ?count=N or defaults to 1
func parseCount(r *http.Request) int {
	if s := r.URL.Query().Get("count"); s != "" {
		if c, err := strconv.Atoi(s); err == nil {
			return c
		}
	}
	return 1
}

// parseStream reads ?stream=1 to enable SSE
func parseStream(r *http.Request) bool {
	return r.URL.Query().Get("stream") == "1"
}

func recipesHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	data, err := os.ReadFile("recipes.json")
	if err != nil {
		http.Error(w, "cannot read recipes.json", http.StatusInternalServerError)
		return
	}

	log.Printf("→ [recipesHandler] loaded %d bytes\n", len(data))
	log.Printf("→ [recipesHandler] preview:\n%s\n", truncate(data, 200))

	w.Write(data)
}

// dfsHandler supports both one-shot and streaming.
func dfsHandler(w http.ResponseWriter, r *http.Request) {
	target := r.URL.Query().Get("target")
	if target == "" {
		http.Error(w, "missing target", http.StatusBadRequest)
		return
	}
	maxRecipes := parseCount(r)
	streaming := parseStream(r)

	log.Printf("→ [dfsHandler] target=%q maxRecipes=%d stream=%v\n", target, maxRecipes, streaming)

	// reset state
	recipe.VisitedMap = make(map[string]*recipe.RecipeTreeNode)

	// build root
	root := &recipe.RecipeTreeNode{Name: target}
	stopChan := make(chan bool)
	wg := &sync.WaitGroup{}
	mu := &sync.Mutex{}
	treeChan := make(chan *recipe.RecipeTreeNode, 10) // buffered
	
	// real metrics
	start := time.Now()
	var nodesVisited int


	// start the search
	wg.Add(1)
	go recipe.BuildRecipeTreeDFS(root, recipe.RecipeMap, maxRecipes, stopChan, wg, mu, &nodesVisited, treeChan)

	if streaming {
		// SSE mode
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}

		// send each partial update as soon as it arrives
		go func() {
			wg.Wait()
			close(treeChan)
		}()

		for node := range treeChan {
			dto := buildDTO(node)
        	elapsed   := time.Since(start).Milliseconds()
        	found     := recipe.CalculateTotalCompleteRecipes(root)

           sse := TreeResponse{
               Tree:         dto,
               TimeTaken:    elapsed,
               NodesVisited: nodesVisited,
               RecipesFound: found,
               MethodUsed:   "DFS",
           }
           data, _ := json.Marshal(sse)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
		return
	}

	// one-shot mode
	go func() {
		wg.Wait()
		close(treeChan)
	}()

	wg.Wait()
	elapsed   := time.Since(start).Milliseconds()
	recipesFound := recipe.CalculateTotalCompleteRecipes(root)	
	recipe.PruneTree(root)
	dto := buildDTO(root)

	resp := TreeResponse{
		Tree:         dto,
		TimeTaken:    elapsed,
		NodesVisited: nodesVisited,
		RecipesFound: recipesFound,
		MethodUsed:   "DFS",
	}
	writeJSON(w, resp)
}

func bfsHandler(w http.ResponseWriter, r *http.Request) {
	target := r.URL.Query().Get("target")
	if target == "" {
		http.Error(w, "missing target", http.StatusBadRequest)
		return
	}
	maxRecipes := parseCount(r)
	streaming := parseStream(r)

	log.Printf("→ [bfsHandler] target=%q maxRecipes=%d stream=%v\n", target, maxRecipes, streaming)

	// reset state
	recipe.VisitedMap = make(map[string]*recipe.RecipeTreeNode)

	// build root
	root := &recipe.RecipeTreeNode{Name: target}
	stopChan := make(chan bool)
	wg := &sync.WaitGroup{}
	mu := &sync.Mutex{}
	treeChan := make(chan *recipe.RecipeTreeNode, 10) // buffered

	// real metrics
	start := time.Now()
	var nodesVisited int
	
	// start the search
	wg.Add(1)
	go recipe.BuildRecipeTreeDFS(root, recipe.RecipeMap, maxRecipes, stopChan, wg, mu, &nodesVisited, treeChan)

	if streaming {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}

		go func() {
			wg.Wait()
			close(treeChan)
		}()

		for node := range treeChan {
			dto := buildDTO(node)
            // compute current metrics
            elapsed   := time.Since(start).Milliseconds()
            found     := recipe.CalculateTotalCompleteRecipes(root)

            sse := TreeResponse{
                Tree:         dto,
                TimeTaken:    elapsed,
                NodesVisited: nodesVisited,
                RecipesFound: found,
                MethodUsed:   "BFS",
            }
            data, _ := json.Marshal(sse)			
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
		return
	}

	// one-shot
	go func() {
		wg.Wait()
		close(treeChan)
	}()

	wg.Wait()
	elapsed := time.Since(start).Milliseconds()
	recipesFound := recipe.CalculateTotalCompleteRecipes(root)
	recipe.PruneTree(root)
	dto := buildDTO(root)

	resp := TreeResponse{
		Tree:         dto,
		TimeTaken:    elapsed,
		NodesVisited: nodesVisited,
		RecipesFound: recipesFound,
		MethodUsed:   "BFS",
	}
	writeJSON(w, resp)
}

func Start() {
	var err error
	recipe.RecipeMap, err = recipe.ReadJson("recipes.json")
	if err != nil {
		log.Fatalf("Failed to load recipes.json: %v", err)
	}

	http.HandleFunc("/api/recipes", recipesHandler)
	http.HandleFunc("/api/dfs", dfsHandler)
	http.HandleFunc("/api/bfs", bfsHandler)
	http.HandleFunc("/api/bidirectional", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Bidirectional not implemented", http.StatusNotImplemented)
	})

	log.Println("Server listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// truncate returns up to `n` bytes of `b` (for logging).
func truncate(b []byte, n int) string {
	if len(b) <= n {
		return string(b)
	}
	return string(b[:n]) + "...(truncated)"
}
