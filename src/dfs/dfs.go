package dfs

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

// Recipe represents an element and its possible recipes
type Recipe struct {
	Name    string     `json:"element"`
	Recipes [][]string `json:"recipes"`
}

// RecipeTreeNode represents a node in the recipe tree
type RecipeTreeNode struct {
	Name     string
	Children [][]*RecipeTreeNode
	mu       sync.Mutex // For safely modifying Children during concurrent operations
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

// RecipeMap is a map of element name to its Recipe
type RecipeMap map[string]Recipe

// IsElemental checks if an element is one of the 4 basic elements
func IsElemental(element string) bool {
	elementals := map[string]bool{
		"water": true,
		"fire":  true,
		"earth": true,
		"air":   true,
	}
	return elementals[element]
}

func (node *RecipeTreeNode) AddChild(child []*RecipeTreeNode) {
	node.mu.Lock()
	defer node.mu.Unlock()
	node.Children = append(node.Children, child)
}

func (node *RecipeTreeNode) GetChildren() [][]*RecipeTreeNode {
	node.mu.Lock()
	defer node.mu.Unlock()
	return node.Children
}

func (node *RecipeTreeNode) SetChildren(children [][]*RecipeTreeNode) {
	node.mu.Lock()
	defer node.mu.Unlock()
	node.Children = children
}

func BuildConcurrentDFS(root *RecipeTreeNode, recipeMap RecipeMap, visited map[string]bool) {

	visited[root.Name] = true

	recipe, ok := recipeMap[root.Name]
	if !ok {
		return
	}

	var wg sync.WaitGroup
	for _, r := range recipe.Recipes {
		wg.Add(1)
		go func(r []string) {
			defer wg.Done()
			for _, childName := range r {
				if visited[childName] {
					continue
				}
				childNode := &RecipeTreeNode{Name: childName}
				root.AddChild([]*RecipeTreeNode{childNode})
				BuildConcurrentDFS(childNode, recipeMap, visited)
			}
		}(r)
	}
	wg.Wait()
}

var VisitedPrintMap = make(map[string]bool)

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
