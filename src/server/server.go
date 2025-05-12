// server.go
package server

import (
    "encoding/json"
    "log"
    "net/http"
    "os"
    "strconv"
    "sync"

    recipe "github.com/Henshou/Tubes2_BE_CraftingTable.git/recipe"
)

// NodeDTO is exactly what the front-end’s Tree.jsx expects.
// We no longer send ImageURL here; the front-end will derive
// its own `/images/${name}.svg` path.
type NodeDTO struct {
    Name    string      `json:"name"`
    Recipes []RecipeDTO `json:"recipes"`
}

type RecipeDTO struct {
    Inputs []NodeDTO `json:"inputs"`
}

// buildDTO walks your in-memory RecipeTreeNode and emits a NodeDTO.
// Note that we initialize all slices to non-nil, so JSON comes out as []
// rather than null.
func buildDTO(node *recipe.RecipeTreeNode) NodeDTO {
    dto := NodeDTO{
        Name:    node.Name,
        Recipes: make([]RecipeDTO, 0),
    }
    for _, group := range node.Children {
        // skip any empty groups
        if len(group) == 0 {
            continue
        }
        inputs := make([]NodeDTO, 0)
        for _, child := range group {
            inputs = append(inputs, buildDTO(child))
        }
        dto.Recipes = append(dto.Recipes, RecipeDTO{Inputs: inputs})
    }
    return dto
}

func writeJSON(w http.ResponseWriter, v interface{}) {
    w.Header().Set("Access-Control-Allow-Origin", "*")
    w.Header().Set("Content-Type", "application/json")
    if err := json.NewEncoder(w).Encode(v); err != nil {
        http.Error(w, "json encode error", http.StatusInternalServerError)
    }
}

func parseCount(r *http.Request) int {
    if s := r.URL.Query().Get("count"); s != "" {
        if c, err := strconv.Atoi(s); err == nil {
            return c
        }
    }
    return 1
}

func recipesHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Access-Control-Allow-Origin", "*")
    w.Header().Set("Content-Type", "application/json")

    data, err := os.ReadFile("recipes.json")
    if err != nil {
        http.Error(w, "cannot read recipes.json", http.StatusInternalServerError)
        return
    }

    // —— DEBUG: print the raw recipes.json to your server log
    log.Printf("→ [recipesHandler] loaded %d bytes from recipes.json\n", len(data))
    log.Printf("→ [recipesHandler] sample payload:\n%s\n", truncate(data, 200))

    w.Write(data)
}

func dfsHandler(w http.ResponseWriter, r *http.Request) {
    target := r.URL.Query().Get("target")
    if target == "" {
        http.Error(w, "missing target", http.StatusBadRequest)
        return
    }
    maxRecipes := parseCount(r)
    log.Printf("→ [dfsHandler] target=%q maxRecipes=%d\n", target, maxRecipes)

    // reset state
    recipe.VisitedMap = make(map[string]*recipe.RecipeTreeNode)

    // build the tree
    root := &recipe.RecipeTreeNode{Name: target}
    stopChan := make(chan bool)
    wg := &sync.WaitGroup{}
    mu := &sync.Mutex{}

    wg.Add(1)
    go recipe.BuildRecipeTreeDFS(root, recipe.RecipeMap, maxRecipes, stopChan, wg, mu)
    wg.Wait()

    // convert to DTO
    dto := buildDTO(root)

    // —— DEBUG: marshal and log the DTO
    if b, err := json.MarshalIndent(dto, "", "  "); err == nil {
        log.Printf("→ [dfsHandler] returning DTO:\n%s\n", b)
    } else {
        log.Printf("!! [dfsHandler] failed to marshal DTO: %v\n", err)
    }

    writeJSON(w, dto)
}

func bfsHandler(w http.ResponseWriter, r *http.Request) {
    target := r.URL.Query().Get("target")
    if target == "" {
        http.Error(w, "missing target", http.StatusBadRequest)
        return
    }
    maxRecipes := parseCount(r)
    log.Printf("→ [bfsHandler] target=%q maxRecipes=%d\n", target, maxRecipes)

    recipe.VisitedMap = make(map[string]*recipe.RecipeTreeNode)

    root := &recipe.RecipeTreeNode{Name: target}
    stopChan := make(chan bool)
    wg := &sync.WaitGroup{}
    mu := &sync.Mutex{}

    wg.Add(1)
    go recipe.BuildRecipeTreeBFS(root, recipe.RecipeMap, maxRecipes, stopChan, wg, mu)
    wg.Wait()

    dto := buildDTO(root)

    // —— DEBUG: marshal and log the DTO
    if b, err := json.MarshalIndent(dto, "", "  "); err == nil {
        log.Printf("→ [bfsHandler] returning DTO:\n%s\n", b)
    } else {
        log.Printf("!! [bfsHandler] failed to marshal DTO: %v\n", err)
    }

    writeJSON(w, dto)
}

// Start hooks up the handlers and loads your recipes.json into memory.
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
