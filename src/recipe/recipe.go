package recipe

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"sync"
)

type Recipe struct {
	Name    string     `json:"element"`
	Recipes [][]string `json:"recipes"`
}

type RecipeTreeNode struct {
	Name     string
	Children [][]*RecipeTreeNode
}

var VisitedMap = make(map[string]*RecipeTreeNode)
var RecipeMap = make(map[string]Recipe) // from readJson
var VisitedPrintMap = make(map[string]bool)

func (node *RecipeTreeNode) AddChild(child []*RecipeTreeNode) {
	node.Children = append(node.Children, child)
}

func ReadJson(filename string) (map[string]Recipe, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %w", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	var recipes []Recipe
	if err := decoder.Decode(&recipes); err != nil {
		return nil, fmt.Errorf("error decoding JSON: %w", err)
	}

	var recipeMap = make(map[string]Recipe)
	for _, recipe := range recipes {
		recipeMap[recipe.Name] = recipe
	}

	return recipeMap, nil
}

var MaxQueueLength int

func BuildRecipeTreeBFS(root *RecipeTreeNode, shortest bool) error {
	queue := []*RecipeTreeNode{root}
	for len(queue) > 0 {
		if len(queue) > MaxQueueLength {
			MaxQueueLength = len(queue)
		}
		current := queue[0]
		queue = queue[1:]
		if _, ok := VisitedMap[current.Name]; ok {
			continue
		}

		recipe := RecipeMap[current.Name]

		if len(recipe.Recipes) == 0 {
			fmt.Println("Base Element Found: ", current.Name)
		}

		for _, child := range recipe.Recipes {
			var Children []*RecipeTreeNode
			for _, Name := range child {
				if existing, ok := VisitedMap[Name]; ok {
					Children = append(Children, existing)
					continue
				}
				childNode := &RecipeTreeNode{Name: Name}
				Children = append(Children, childNode)
				queue = append(queue, childNode)
			}
			current.AddChild(Children)
		}
		VisitedMap[current.Name] = current
	}
	return nil
}
func printQueue(queue []*RecipeTreeNode) {
		fmt.Print("Queue: [")
		for _, node := range queue {
			fmt.Print(node.Name, " ")
		}
		fmt.Print("]")
		fmt.Println()
	}

	func SetChildren(node *RecipeTreeNode, children [][]*RecipeTreeNode) {
		node.Children = children
	}

func BuildRecipeTreeDFS(root *RecipeTreeNode, recipeMap map[string]Recipe) {
	// Check if the node has already been visited
	if _, ok := VisitedMap[root.Name]; ok {
		return
	}
	VisitedMap[root.Name] = root

	// Get the recipes for the current element
	recipe, exists := recipeMap[root.Name]
	if !exists {
		return
	}

	var children [][]*RecipeTreeNode
	for _, r := range recipe.Recipes {
		var childNodes []*RecipeTreeNode
		for _, name := range r {
			childNode := &RecipeTreeNode{Name: name}
			childNodes = append(childNodes, childNode)
			BuildRecipeTreeDFS(childNode, recipeMap)
		}
		children = append(children, childNodes)
	}
	root.Children = append(root.Children, children...)
}

func BuildRecipeTreeDFSConcurrent(root *RecipeTreeNode, recipeMap map[string]Recipe, wg *sync.WaitGroup, mu *sync.Mutex) {
	if wg != nil {
		defer wg.Done()
	}

	// Use mutex to protect the shared VisitedMap
	mu.Lock()
	if _, ok := VisitedMap[root.Name]; ok {
		mu.Unlock()
		return
	}
	VisitedMap[root.Name] = root
	mu.Unlock()

	// Get the recipes for the current element
	recipe, exists := recipeMap[root.Name]
	if !exists {
		return
	}

	var children [][]*RecipeTreeNode
	childWg := &sync.WaitGroup{}

	for _, r := range recipe.Recipes {
		var childNodes []*RecipeTreeNode
		for _, name := range r {
			childNode := &RecipeTreeNode{Name: name}
			childNodes = append(childNodes, childNode)

			// Launch a goroutine for each child
			childWg.Add(1)
			go BuildRecipeTreeDFSConcurrent(childNode, recipeMap, childWg, mu)
		}
		children = append(children, childNodes)
	}

	// Wait for all child goroutines to complete
	childWg.Wait()
	SetChildren(root, children)
}

func BuildRecipeTreeConcurrentBFS(target string) *RecipeTreeNode {
	const workerCount = 5
	queue := make(chan *RecipeTreeNode, 1000)
	var wg sync.WaitGroup
	var mutex sync.Mutex

	root := &RecipeTreeNode{Name: target}
	queue <- root

	VisitedMap[target] = root

	// Channel to signal when all workers are done
	done := make(chan struct{})

	// Start worker goroutines
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for current := range queue {
				// Get recipe for current element
				recipe, ok := RecipeMap[current.Name]
				if !ok || len(recipe.Recipes) == 0 {
					continue // base element
				}

				for _, group := range recipe.Recipes {
					var children []*RecipeTreeNode

					for _, childName := range group {
						mutex.Lock()
						childNode, exists := VisitedMap[childName]
						if !exists {
							childNode = &RecipeTreeNode{Name: childName}
							VisitedMap[childName] = childNode
							queue <- childNode // enqueue safely
						}
						mutex.Unlock()

						children = append(children, childNode)
					}

					mutex.Lock()
					current.AddChild(children)
					mutex.Unlock()
				}
			}
		}()
	}

	// Wait for workers and close 'done'
	go func() {
		wg.Wait()
		close(done)
	}()

	// Wait until work finishes
	<-done
	return root
}

func PrintRecipeTree(node *RecipeTreeNode, indent string) {
	if node == nil {
		return
	}

	fmt.Println(indent + "- " + node.Name)

	// Print each alternative recipe path
	for i, alternative := range node.Children {
		if len(alternative) > 0 {
			fmt.Println(indent + "  Recipe alternative #" + strconv.Itoa(i+1) + ":")
			for _, child := range alternative {
				PrintRecipeTree(child, indent+"    ")
			}
		}
	}
}

func HasBaseElements(node *RecipeTreeNode) bool {
	if len(node.Children) == 0 {
		return true
	}
	for _, group := range node.Children {
		for _, child := range group {
			if HasBaseElements(child) {
				return true
			}
		}
	}
	return false
}

func BuildConcurrentDFS(root *RecipeTreeNode, wg *sync.WaitGroup, visited *sync.Map) {
	defer wg.Done() // Ensure Done() is called even if we panic

	// Check if already visited (thread-safe)
	if _, loaded := visited.LoadOrStore(root.Name, struct{}{}); loaded {
		fmt.Println("Already visited:", root.Name)
		return
	}

	recipe, ok := RecipeMap[root.Name]
	if !ok {
		return
	}

	if len(recipe.Recipes) == 0 {
		fmt.Println("Base Element Found:", root.Name)
		return
	}

	var wgChildren sync.WaitGroup
	for _, childGroup := range recipe.Recipes {
		for _, childName := range childGroup {
			wgChildren.Add(1) // Increment BEFORE goroutine
			childNode := &RecipeTreeNode{Name: childName}
			root.AddChild([]*RecipeTreeNode{childNode})

			go func(node *RecipeTreeNode) {
				defer wgChildren.Done() // Defer Done() inside goroutine
				BuildConcurrentDFS(node, &wgChildren, visited)
			}(childNode)
		}
	}
	wgChildren.Wait() // Wait for all children
}

func BuildTreeWithLimit(target string, recipeMap map[string]Recipe, limit int) *RecipeTreeNode {
	// Inisialisasi root
	root := &RecipeTreeNode{Name: target}

	// Jalankan DFS concurrent
	var wg sync.WaitGroup
	var mu sync.Mutex
	wg.Add(1)
	BuildRecipeTreeDFSConcurrent(root, recipeMap, &wg, &mu)
	wg.Wait()

	return root
}
