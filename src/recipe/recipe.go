package recipe

import (
	"encoding/json"
	"fmt"
	"os"
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

func BuildRecipeTreeBFS(root *RecipeTreeNode) error {
	queue := []*RecipeTreeNode{root}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if _, ok := VisitedMap[current.Name]; ok {
			continue
		}

		recipe, ok := RecipeMap[current.Name]
		if !ok {
			return fmt.Errorf("recipe not found: %s", current.Name)
		}
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

	// Flatten all children combinations
	var allChildren []*RecipeTreeNode
	for _, group := range node.Children {
		allChildren = append(allChildren, group...)
	}

	// Recursively print each child
	for i, child := range allChildren {
		PrintRecipeTree(child, prefix, i == len(allChildren)-1)
		VisitedPrintMap[node.Name] = true
	}
}
