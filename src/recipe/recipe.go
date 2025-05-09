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


func BuildRecipeTreeDFSConcurrent(
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
			go BuildRecipeTreeDFSConcurrent(childNode, recipeMap, childWg, mu, maxRecipes, validRecipes)
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

func BuildRecipeTreeBFSConcurrent(
	root *RecipeTreeNode,
	recipeMap map[string]Recipe,
	maxRecipes int,
	validRecipes *[]string,
) {
	var mu sync.Mutex
	queue := []*RecipeTreeNode{root}

	for len(queue) > 0 && len(*validRecipes) < maxRecipes {
		nextQueue := []*RecipeTreeNode{}
		var wg sync.WaitGroup

		for _, node := range queue {
			wg.Add(1)

			go func(n *RecipeTreeNode) {
				defer wg.Done()

				mu.Lock()
				if _, visited := VisitedMap[n.Name]; visited {
					mu.Unlock()
					return
				}
				VisitedMap[n.Name] = n
				mu.Unlock()

				recipe, exists := recipeMap[n.Name]
				if !exists {
					return
				}

				var children [][]*RecipeTreeNode

				for _, r := range recipe.Recipes {
					var childNodes []*RecipeTreeNode
					for _, name := range r {
						childNode := &RecipeTreeNode{Name: name}
						childNodes = append(childNodes, childNode)

						mu.Lock()
						// Add to next level if not visited yet
						if _, seen := VisitedMap[name]; !seen {
							nextQueue = append(nextQueue, childNode)
						}
						mu.Unlock()
					}

					// Check if recipe is valid (2 base elements)
					if len(r) == 2 && IsBaseElement(r[0]) && IsBaseElement(r[1]) {
						mu.Lock()
						if len(*validRecipes) < maxRecipes {
							*validRecipes = append(*validRecipes, fmt.Sprintf("%s = %s + %s", n.Name, r[0], r[1]))
						}
						mu.Unlock()
					}

					children = append(children, childNodes)
				}

				SetChildren(n, children)
			}(node)
		}

		wg.Wait()
		queue = nextQueue
	}
}


func SetChildren(node *RecipeTreeNode, children [][]*RecipeTreeNode) {
	node.Children = children
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

