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
				if exists && ingredientRecipe.Tier >= recipe.Tier {
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
		if len(group) != 2 {
			continue // Skip if not a valid recipe group
		}
		leftCount := CalculateTotalCompleteRecipes(group[0])
		rightCount := CalculateTotalCompleteRecipes(group[1])
		if leftCount > 0 && rightCount > 0 {
			total += leftCount * rightCount
		}
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

func BuildRecipeTreeDFS(
	root *RecipeTreeNode,
	recipeMap map[string]Recipe,
	maxRecipes int,
	stopChan chan bool, // Channel to signal stopping
	wg *sync.WaitGroup, // WaitGroup for goroutines
	mu *sync.Mutex, // Mutex to safely modify shared variables
) {
	defer wg.Done()

	// Queue for BFS, starting from the root node
	stack := []*RecipeTreeNode{root}

	// Loop while there are nodes in the queue and the number of valid recipes is less than maxRecipes
	for len(stack) > 0 {
		select {
		case <-stopChan:
			return // Stop further search if signal is received
		default:
		}
		node := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		recipe, exists := recipeMap[node.Name]
		if !exists {
			continue
		}

		var children [][]*RecipeTreeNode
		var childWg sync.WaitGroup // To wait for child goroutines

		// Iterate through each recipe for the current node
		for _, r := range recipe.Recipes {
			childWg.Add(1)

			// Goroutine to process each recipe path concurrently
			go func(r []string) {
				defer childWg.Done()

				var childNodes []*RecipeTreeNode
				for _, name := range r {
					childNode := &RecipeTreeNode{Name: name}
					childNodes = append(childNodes, childNode)

					mu.Lock()
					stack = append(stack, childNode)
					mu.Unlock()
				}

				// Check if the recipe is valid (both children must be base elements)
				if len(r) == 2 && IsBaseElement(r[0]) && IsBaseElement(r[1]) {
					for _, name := range stack {
						fmt.Print(name.Name, " ")
					}
					mu.Lock()
					if CalculateTotalCompleteRecipes(root) >= maxRecipes {
						stopChan <- true
						return
					}
					mu.Unlock()
				}

				// Add the children for this recipe to the children list
				mu.Lock()
				children = append(children, childNodes)
				mu.Unlock()

			}(r)
		}

		// Wait for all goroutines to finish processing the current node's recipes
		childWg.Wait()

		// Set the children for this node
		mu.Lock()
		SetChildren(node, children)
		mu.Unlock()

		// Check if stop signal was received
		select {
		case <-stopChan:
			return // Stop further search if signal is received
		default:
			// Continue processing if no stop signal
		}
	}
}

func BuildRecipeTreeBFS(
	root *RecipeTreeNode,
	recipeMap map[string]Recipe,
	maxRecipes int,
	stopChan chan bool, // Channel to signal stopping
	wg *sync.WaitGroup, // WaitGroup for goroutines
	mu *sync.Mutex, // Mutex to safely modify shared variables
) {
	defer wg.Done()

	// Queue for BFS, starting from the root node
	queue := []*RecipeTreeNode{root}

	// Loop while there are nodes in the queue and the number of valid recipes is less than maxRecipes
	for len(queue) > 0 {
		select {
		case <-stopChan:
			return // Stop further search if signal is received
		default:
		}
		// Process the first node in the queue
		node := queue[0]
		queue = queue[1:]

		recipe, exists := recipeMap[node.Name]
		if !exists {
			continue
		}

		var children [][]*RecipeTreeNode
		var childWg sync.WaitGroup // To wait for child goroutines

		// Iterate through each recipe for the current node
		for _, r := range recipe.Recipes {
			childWg.Add(1)

			// Goroutine to process each recipe path concurrently
			go func(r []string) {
				defer childWg.Done()

				var childNodes []*RecipeTreeNode
				for _, name := range r {
					childNode := &RecipeTreeNode{Name: name}
					childNodes = append(childNodes, childNode)

					// Add to the next level (queue) for future expansion
					mu.Lock()
					queue = append(queue, childNode)
					mu.Unlock()
				}

				// Check if the recipe is valid (both children must be base elements)
				if len(r) == 2 && IsBaseElement(r[0]) && IsBaseElement(r[1]) {
					mu.Lock()
					// Stop the search if we reach maxRecipes
					if CalculateTotalCompleteRecipes(root) >= maxRecipes {
						stopChan <- true // Send stop signal
						return
					}
					mu.Unlock()
				}

				// Add the children for this recipe to the children list
				mu.Lock()
				children = append(children, childNodes)
				mu.Unlock()

			}(r)
		}

		// Wait for all goroutines to finish processing the current node's recipes
		childWg.Wait()

		// Set the children for this node
		mu.Lock()
		SetChildren(node, children)
		mu.Unlock()

		// Check if stop signal was received
		select {
		case <-stopChan:
			return // Stop further search if signal is received
		default:
			// Continue processing if no stop signal
		}
	}
}

func StopSearch(stopChan chan bool, wg *sync.WaitGroup) {
	defer wg.Done()
	<-stopChan
	// Stop the search when the signal is received
	fmt.Println("Stopping the search!")
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

func PruneTree(node *RecipeTreeNode) {
	var newChildren [][]*RecipeTreeNode
	for _, recipe := range node.Children {
		bothBase := true
		for _, child := range recipe {
			if CalculateTotalCompleteRecipes(child) == 0 {
				bothBase = false
				fmt.Println("Pruning", child.Name)
				break
			}
		}
		if bothBase {
			newChildren = append(newChildren, recipe)
		}
	}
	SetChildren(node, newChildren)
	for _, recipe := range newChildren {
		for _, child := range recipe {
			PruneTree(child)
		}
	}
}
