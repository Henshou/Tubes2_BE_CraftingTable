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
	Tier    int        `json:"tier"`
}

type RecipeTreeNode struct {
	Name     string
	Children [][]*RecipeTreeNode
}

// Global maps for tracking visited nodes and recipe data
var VisitedMap = make(map[string]*RecipeTreeNode)
var RecipeMap = make(map[string]Recipe) // from readJson
var CompletedRecipes = make(map[string]int)

// Check if an element is a base element (leaf node)
func IsBaseElement(name string) bool {
	recipe, exists := RecipeMap[name]
	if !exists {
		return false
	}
	if recipe.Tier == 0 {
		return true
	}
	return false
}

func IsBaseElementRecipe(Recipe Recipe) bool {
	return Recipe.Tier == 0
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

	recipeMap := make(map[string]Recipe)

	for _, recipe := range recipes {
		recipeMap[recipe.Name] = recipe
	}

	for name, recipe := range recipeMap {
		validRecipes := [][]string{}

		for _, ingredients := range recipe.Recipes {
			valid := true

			for _, ingredient := range ingredients {
				ingredientRecipe, exists := recipeMap[ingredient]
				if exists && ingredientRecipe.Tier > recipe.Tier {
					valid = false
					break
				}
			}

			if valid {
				validRecipes = append(validRecipes, ingredients)
			}
		}

		if len(validRecipes) > 0 {
			recipe.Recipes = validRecipes
			recipeMap[name] = recipe
		} else {
			delete(recipeMap, name)
		}
	}

	return recipeMap, nil
}

func CalculateTotalCompleteRecipes(root *RecipeTreeNode) int {
	if root == nil {
		return 0
	}

	if IsBaseElement(root.Name) {
		return 1
	}

	total := 0
	for _, group := range root.Children {
		leftCount := CalculateTotalCompleteRecipes(group[0])
		rightCount := CalculateTotalCompleteRecipes(group[1])
		if leftCount > 0 && rightCount > 0 {
			total += leftCount * rightCount
		}
	}
	if total > 0 {
		CompletedRecipes[root.Name] = total
	}
	return total
}

func IsCompleteRecipe(recipe Recipe) bool {
	if len(recipe.Recipes) == 0 {
		return false
	}
	for _, r := range recipe.Recipes {
		if len(r) != 2 {
			return false
		}
		if !IsBaseElement(r[0]) || !IsBaseElement(r[1]) {
			return false
		}
	}
	return true
}

func BuildRecipeTreeDFSConcurrent(
	root *RecipeTreeNode,
	recipeMap map[string]Recipe,
	wg *sync.WaitGroup,
	mu *sync.Mutex,
	maxRecipes int,
	validRecipes *[]string,
) {
	if wg != nil {
		defer wg.Done()
	}

	// Use mutex to protect the shared VisitedMap

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
func BuildRecipeTreeBFS(
	root *RecipeTreeNode,
	recipeMap map[string]Recipe,
	maxRecipes int,
	validRecipes *[]string,
	stopChan chan bool, // Channel to signal stopping
	wg *sync.WaitGroup, // WaitGroup for goroutines
	mu *sync.Mutex, // Mutex to safely modify shared variables
) {
	defer wg.Done()

	queue := []*RecipeTreeNode{root}

	// Loop while the queue has nodes and the number of valid recipes is less than maxRecipes
	for len(queue) > 0 {
		// Process the first node in the queue
		node := queue[0]
		queue = queue[1:]

		recipe, exists := recipeMap[node.Name]
		if !exists {
			continue
		}

		var children [][]*RecipeTreeNode

		// Iterate through each recipe for the current node
		for _, r := range recipe.Recipes {
			var childNodes []*RecipeTreeNode

			// Expand the tree for this recipe
			for _, name := range r {
				childNode := &RecipeTreeNode{Name: name}
				childNodes = append(childNodes, childNode)

				// Add to the next level (queue) for future expansion
				queue = append(queue, childNode)
			}

			// Check if the recipe is valid (both children must be base elements)
			if len(r) == 2 && IsBaseElement(r[0]) && IsBaseElement(r[1]) {
				// Stop the search if we reach maxRecipes
				mu.Lock()
				if CalculateTotalCompleteRecipes(root) >= maxRecipes {
					mu.Unlock()
					stopChan <- true // Send stop signal
					return
				}
				// Add the valid recipe to the list
				*validRecipes = append(*validRecipes, fmt.Sprintf("%s = %s + %s", node.Name, r[0], r[1]))
				mu.Unlock()
			}

			// Add the children for this node to the children list
			children = append(children, childNodes)
		}

		// Set the children for this node
		SetChildren(node, children)

		// Check if stop signal was received
		select {
		case <-stopChan:
			return // Stop further search
		default:
			// Continue processing if no stop signal
		}
	}
}

func StopSearch(stopChan chan bool) {
	<-stopChan
	// Stop the search when the signal is received
	fmt.Println("Stopping the search!")
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
