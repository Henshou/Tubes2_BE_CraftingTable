package recipe

import (
	"encoding/json"
	"fmt"
	"os"
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

func BuildRecipeTreeDFS(root *RecipeTreeNode) error {
	stack := []*RecipeTreeNode{root}

	for len(stack) > 0 {
		n := len(stack) - 1
		current := stack[n]
		stack = stack[:n]

		if _, ok := VisitedMap[current.Name]; ok {
			continue
		}

		recipe, ok := RecipeMap[current.Name]
		if !ok {
			return fmt.Errorf("recipe not found: %s", current.Name)
		}

		if len(recipe.Recipes) == 0 {
			fmt.Println("Base Element Found:", current.Name)
		}

		for _, childGroup := range recipe.Recipes {
			var children []*RecipeTreeNode
			for _, childName := range childGroup {
				if existing, ok := VisitedMap[childName]; ok {
					children = append(children, existing)
				} else {
					childNode := &RecipeTreeNode{Name: childName}
					children = append(children, childNode)
					stack = append(stack, childNode)
				}
			}
			current.AddChild(children)
		}

		VisitedMap[current.Name] = current
	}
	return nil
}

func PrintRecipeTree(node *RecipeTreeNode, prefix string, isLast bool) {
	// Print the current node
	if _, ok := VisitedPrintMap[node.Name]; ok {
		return
	}
	fmt.Print(prefix)
	if isLast {
		fmt.Print("└── ")
		prefix += "    "
	} else {
		fmt.Print("├── ")
		prefix += "│   "
	}
	fmt.Println(node.Name)

	var allChildren []*RecipeTreeNode
	for _, group := range node.Children {
		allChildren = append(allChildren, group...)
	}

	for i, child := range allChildren {
		PrintRecipeTree(child, prefix, i == len(allChildren)-1)
		VisitedPrintMap[node.Name] = true
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
