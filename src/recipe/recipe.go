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
	var Recipes map[string]Recipe
	if err := decoder.Decode(&Recipes); err != nil {
		return nil, fmt.Errorf("error decoding JSON: %w", err)
	}

	return Recipes, nil
}

func BuildRecipeTreeBFS(root *RecipeTreeNode) error {
	queue := []*RecipeTreeNode{root}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if _, ok := VisitedMap[current.Name]; ok {
			continue
		}

		VisitedMap[current.Name] = current
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
	}
	return nil
}
