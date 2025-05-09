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

// Global maps for tracking visited nodes and recipe data
var VisitedMap = make(map[string]*RecipeTreeNode)
var RecipeMap = make(map[string]Recipe) // from readJson

// Check if an element is a base element (leaf node)
func IsBaseElement(name string) bool {
	baseElements := []string{
		"Air", "Earth", "Fire", "Water"}
	for _, base := range baseElements {
		if name == base {
			return true
		}
	}
	return false
}

func BuildRecipeTreeBFS(root *RecipeTreeNode, maxRecipes int) ([]string, error) {
	// Queue for BFS
	queue := []*RecipeTreeNode{root}
	var validRecipes []string

	for len(queue) > 0 {
		// Dequeue the next node
		current := queue[0]
		queue = queue[1:]

		// Skip if already visited
		if _, ok := VisitedMap[current.Name]; ok {
			continue
		}

		// Mark the node as visited
		VisitedMap[current.Name] = current

		// Get the recipe for the current element
		recipe, exists := RecipeMap[current.Name]
		if !exists {
			continue
		}

		// Iterate over all recipes of the current element
		for _, recipeCombination := range recipe.Recipes {
			// Check if the current recipe consists of exactly 2 base elements
			if len(recipeCombination) == 2 &&
				IsBaseElement(recipeCombination[0]) && IsBaseElement(recipeCombination[1]) {
				// Valid complete recipe found
				validRecipes = append(validRecipes, fmt.Sprintf("%s = %s + %s", current.Name, recipeCombination[0], recipeCombination[1]))

				// Stop if we have found the maximum number of recipes
				if len(validRecipes) >= maxRecipes {
					return validRecipes, nil
				}
			}
		}

		// Add child nodes to the queue for all recipes
		for _, child := range recipe.Recipes {
			var children []*RecipeTreeNode
			for _, childName := range child {
				// Create child nodes and enqueue them
				if existing, ok := VisitedMap[childName]; ok {
					children = append(children, existing)
				} else {
					childNode := &RecipeTreeNode{Name: childName}
					queue = append(queue, childNode)
					children = append(children, childNode)
				}
			}
			current.Children = append(current.Children, children)
		}
	}

	return validRecipes, nil
}

// Build the recipe tree using DFS, and stop when a complete recipe is found
func BuildRecipeTreeDFS(root *RecipeTreeNode, maxRecipes int) ([]string, error) {
	var validRecipes []string

	// DFS helper function to explore the tree
	var dfs func(node *RecipeTreeNode)
	dfs = func(node *RecipeTreeNode) {
		if len(validRecipes) >= maxRecipes {
			return
		}

		// Skip if already visited
		if _, ok := VisitedMap[node.Name]; ok {
			return
		}

		// Mark the node as visited
		VisitedMap[node.Name] = node

		// Get the recipe for the current element
		recipe, exists := RecipeMap[node.Name]
		if !exists {
			return
		}

		// Iterate over all recipes of the current element
		for _, recipeCombination := range recipe.Recipes {
			// Check if the current recipe consists of exactly 2 base elements
			if len(recipeCombination) == 2 &&
				IsBaseElement(recipeCombination[0]) && IsBaseElement(recipeCombination[1]) {
				// Valid complete recipe found
				validRecipes = append(validRecipes, fmt.Sprintf("%s = %s + %s", node.Name, recipeCombination[0], recipeCombination[1]))

				// Stop if we have found the maximum number of recipes
				if len(validRecipes) >= maxRecipes {
					return
				}
			}
		}

		// Recursively explore the children for all recipes
		for _, recipeCombination := range recipe.Recipes {
			var children []*RecipeTreeNode
			for _, childName := range recipeCombination {
				childNode := &RecipeTreeNode{Name: childName}
				dfs(childNode)
				children = append(children, childNode)
			}
			node.Children = append(node.Children, children)
		}
	}

	// Start DFS from the root node
	dfs(root)
	return validRecipes, nil
}

// Function to read the JSON data and load the recipes
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

	recipeMap := make(map[string]Recipe)
	for _, recipe := range recipes {
		recipeMap[recipe.Name] = recipe
	}

	return recipeMap, nil
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

func BuildRecipeTreeDFSConcurrents(
	root *RecipeTreeNode,
	recipeMap map[string]Recipe,
	wg *sync.WaitGroup,
	mu *sync.Mutex,
	maxRecipes int, // Stop after finding maxRecipes valid recipes
	validRecipes *[]string, // Store the valid recipes found
) {
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

	// Iterate over all recipes for the current element
	for _, r := range recipe.Recipes {
		var childNodes []*RecipeTreeNode
		for _, name := range r {
			childNode := &RecipeTreeNode{Name: name}
			childNodes = append(childNodes, childNode)

			// Launch a goroutine for each child
			childWg.Add(1)
			go BuildRecipeTreeDFSConcurrents(childNode, recipeMap, childWg, mu, maxRecipes, validRecipes)
		}

		// After launching child goroutines, check if the current recipe is valid
		if len(r) == 2 && IsBaseElement(r[0]) && IsBaseElement(r[1]) {
			// Valid complete recipe found
			if len(*validRecipes) < maxRecipes {
				// Valid complete recipe found
				mu.Lock() // Lock to safely append to validRecipes
				*validRecipes = append(*validRecipes, fmt.Sprintf("%s = %s + %s", root.Name, r[0], r[1]))
				mu.Unlock()
			}

			// Stop if we have found the maximum number of recipes
			if len(*validRecipes) >= maxRecipes {
				childWg.Wait() // Wait for all child goroutines to finish
				return
			}
		}

		children = append(children, childNodes)
	}

	// Wait for all child goroutines to complete
	childWg.Wait()

	// After the child goroutines finish, assign the children to the current node
	SetChildren(root, children)

	// Optionally, you can check for valid recipes here and append them to `validRecipes`
	// This can be based on your recipe validation criteria
	if len(*validRecipes) >= maxRecipes {
		return
	}
}

func SetChildren(node *RecipeTreeNode, children [][]*RecipeTreeNode) {
	node.Children = children
}

func worker(queue chan *RecipeTreeNode, validRecipes *[]string, wg *sync.WaitGroup, mu *sync.Mutex, maxRecipes int, stopCh chan struct{}, workerID int) {
	defer wg.Done()

	fmt.Printf("Worker %d starting\n", workerID)

	for {
		select {
		case node, ok := <-queue:
			// If the queue is closed and empty, exit
			if !ok {
				fmt.Printf("Worker %d exiting: queue closed\n", workerID)
				return
			}

			// If stop signal is received, immediately exit
			select {
			case <-stopCh:
				// Debug: Stop signal received, worker exits
				fmt.Printf("Worker %d received stop signal, stopping\n", workerID)
				return
			default:
			}

			// Skip if already visited
			mu.Lock()
			if _, ok := VisitedMap[node.Name]; ok {
				mu.Unlock()
				continue
			}
			// Mark the node as visited
			VisitedMap[node.Name] = node
			mu.Unlock()

			// Get the recipe for the current element
			recipe, exists := RecipeMap[node.Name]
			if !exists {
				continue
			}

			// Process the recipes for the current element
			for _, recipeCombination := range recipe.Recipes {
				// Check if the current recipe consists of exactly 2 base elements
				if len(recipeCombination) == 2 &&
					IsBaseElement(recipeCombination[0]) && IsBaseElement(recipeCombination[1]) {
					// Valid complete recipe found
					mu.Lock()
					*validRecipes = append(*validRecipes, fmt.Sprintf("%s = %s + %s", node.Name, recipeCombination[0], recipeCombination[1]))
					// Debug: Log valid recipe found
					fmt.Printf("Worker %d found valid recipe: %s = %s + %s\n", workerID, node.Name, recipeCombination[0], recipeCombination[1])

					// Stop if we have found the maximum number of recipes
					if len(*validRecipes) >= maxRecipes {
						mu.Unlock()
						// Signal all workers to stop
						stopCh <- struct{}{}
						fmt.Printf("Worker %d stopping: reached maxRecipes\n", workerID)
						return
					}
					mu.Unlock()
				}
			}

			// Add child nodes to the queue for further processing
			for _, child := range recipe.Recipes {
				for _, childName := range child {
					// Skip if it's already visited
					mu.Lock()
					if _, ok := VisitedMap[childName]; ok {
						mu.Unlock()
						continue
					}
					childNode := &RecipeTreeNode{Name: childName}
					mu.Unlock()

					// Check for stop signal before adding child node
					select {
					case <-stopCh:
						// If stop signal received, break the loop
						fmt.Printf("Worker %d received stop signal while adding child node\n", workerID)
						return
					default:
					}

					// Enqueue the child node for processing
					queue <- childNode
					// Debug: Log adding child node to queue
					fmt.Printf("Worker %d added child node: %s\n", workerID, childName)
				}
			}

		case <-stopCh:
			// If stop signal received, break the loop
			fmt.Printf("Worker %d received stop signal\n", workerID)
			return
		}
	}
}

// Concurrent BFS with fixed number of workers
func BuildRecipeTreeBFSConcurrents(root *RecipeTreeNode, maxRecipes int) ([]string, error) {
	// Queue for BFS - channel to safely enqueue and dequeue elements
	queue := make(chan *RecipeTreeNode, 100)
	stopCh := make(chan struct{}) // Channel to stop the workers when maxRecipes is reached

	var validRecipes []string
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Start with the root node
	queue <- root

	// Start 5 workers
	numWorkers := 5
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go worker(queue, &validRecipes, &wg, &mu, maxRecipes, stopCh, i+1) // Pass worker ID for debugging
	}

	// Wait for the workers to finish processing
	wg.Wait()

	// Close the queue after all workers are done
	close(queue)
	// Debug: Log queue closure
	fmt.Println("Main function: queue closed")

	// After all workers have finished, return the valid recipes
	return validRecipes, nil
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

func (node *RecipeTreeNode) AddChild(children []*RecipeTreeNode) {
	node.Children = append(node.Children, children)
}

func IsRecipeRoot(node *RecipeTreeNode) bool {
	if node == nil {
		return false
	}
	if len(node.Children) == 0 {
		return true
	}
	for _, group := range node.Children {
		for _, child := range group {
			if IsBaseElement(child.Name) {
				return true
			}
		}
	}
	return false
}

func ValidRecipes(combination []string) int {
	if len(combination) != 2 {
		return 0
	}
	if IsBaseElement(combination[0]) && IsBaseElement(combination[1]) {
		return 0
	}
	baseIndex := -1
	if IsBaseElement(combination[0]) {
		baseIndex = 1
	}
	if IsBaseElement(combination[1]) {
		baseIndex = 0
	}
	if baseIndex == -1 {
		return 0
	}
	// //traverse
	Recipes := 0
	RecipeBase := RecipeMap[combination[baseIndex]]
	println("Base Element:", RecipeBase.Name)
	for _, recipe := range RecipeBase.Recipes {
		if IsBaseElement(recipe[0]) && IsBaseElement(recipe[1]) {
			Recipes++
		}
	}
	return Recipes
	// return 1
}

func BuildRecipeTreeBFSw(root *RecipeTreeNode) (*RecipeTreeNode, int) {
	queue := []*RecipeTreeNode{root}
	visited := map[string]bool{}
	uniqueRecipes := 0

	// Process queue secara BFS
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		recipe, ok := RecipeMap[current.Name]
		if !ok {
			continue // Jika elemen tidak ada dalam RecipeMap, lanjutkan
		}

		if visited[current.Name] {
			for _, group := range recipe.Recipes {
				uniqueRecipes += ValidRecipes(group)
			}
			continue
		}
		visited[current.Name] = true

		// Tambahkan semua anak-anak berdasarkan resep yang ditemukan
		for _, combination := range recipe.Recipes {
			uniqueRecipes += ValidRecipes(combination)
			var children []*RecipeTreeNode
			for _, ingredient := range combination {
				if visited[ingredient] {
					if childNode, ok := VisitedMap[ingredient]; ok {
						children = append(children, childNode)
					}
					continue
				}
				childNode := &RecipeTreeNode{Name: ingredient}
				children = append(children, childNode)
				queue = append(queue, childNode)
			}
			current.Children = append(current.Children, children)
		}
	}

	// Jika pencarian selesai dan tidak menemukan tree yang lengkap
	return nil, uniqueRecipes
}
