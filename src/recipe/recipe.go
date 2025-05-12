package recipe

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"
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
	nodesVisited *int,
	treeChan chan *RecipeTreeNode,
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
					mu.Lock()
					treeChan <- root
					time.Sleep(500 * time.Millisecond) // Simulate some processing time
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
		mu.Lock()
		*nodesVisited++
		mu.Unlock()
	}
}

func BuildRecipeTreeBFS(
	root *RecipeTreeNode,
	recipeMap map[string]Recipe,
	maxRecipes int,
	stopChan chan bool, // Channel to signal stopping
	wg *sync.WaitGroup, // WaitGroup for goroutines
	mu *sync.Mutex, // Mutex to safely modify shared variables
	nodesVisited *int,
	treeChan chan *RecipeTreeNode,
) {
	defer wg.Done()

	// Queue for BFS, starting from the root node
	queue := []*RecipeTreeNode{root}
	// Loop while there are nodes in the queue and the number of valid recipes is less than maxRecipes
	for len(queue) > 0 {
		// for _, node := range queue {
		// 	fmt.Print(node.Name, " ")
		// }
		// fmt.Println()
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
					treeChan <- root
					time.Sleep(500 * time.Millisecond) // Simulate some processing time
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
		mu.Lock()
		*nodesVisited++
		mu.Unlock()
	}
}

func StopSearch(stopChan chan bool, wg *sync.WaitGroup) {
	defer wg.Done()
	<-stopChan
	// Stop the search when the signal is received
	fmt.Println("Stopping the search!")
	time.Sleep(1 * time.Second)
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

// func BidirectionalBFS(
// 	root *RecipeTreeNode,
// 	recipeMap map[string]Recipe,
// 	recipeToTree map[string]*RecipeTreeNode,
// 	maxRecipes int,
// 	stopChan chan bool, // Channel to signal stopping
// 	wg *sync.WaitGroup, // WaitGroup for goroutines
// 	mu *sync.Mutex, // Mutex to safely modify shared variables
// ) {
// 	meets := 0
// 	owned := make(map[string]bool)
// 	tier := 0
// 	// do bfs on the root simultaneously with making the recipe from the ground up
// 	// queue for bfs
// 	queue := []*RecipeTreeNode{root}
// 	recipeQueue := []*RecipeTreeNode{}
// 	for _, base := range GetAllElements(tier) {
// 		owned[base] = true
// 		temp := &RecipeTreeNode{Name: base}
// 		recipeQueue = append(recipeQueue, temp)

// 	}
// }

func BuildRecipeTreeBidirectional(
	root *RecipeTreeNode,
	recipeMap map[string]Recipe,
	recipeToTree map[string]*RecipeTreeNode,
	maxRecipes int,
	nodesVisited *int,
) {
	bfsVisited := make(map[string]*RecipeTreeNode)
	recipe := recipeMap[root.Name]
	highestTier := recipe.Tier
	owned := make(map[string]bool)
	queue := []*RecipeTreeNode{root}
	recipeQueue := []string{}
	for _, base := range GetAllElements(highestTier) {
		temp := &RecipeTreeNode{Name: base}
		recipeToTree[base] = temp
		recipeQueue = append(recipeQueue, base)
	}
	for len(recipeQueue) > 0 && !OwnAllTier(highestTier, owned) && len(queue) > 0 {
		recipeName := recipeQueue[0]
		recipeQueue = recipeQueue[1:]

		if owned[recipeName] {
			continue
		}

		if IsBaseElement(recipeName) {
			owned[recipeName] = true
			canMake := GetCreatedBy(recipeName)
			for _, r := range canMake {
				if owned[r] {
					continue
				}
				recipeQueue = append(recipeQueue, r)
			}
			continue
		}

		if !CanMakeRecipe(recipeName, owned) {
			if !contains(recipeQueue, recipeName) {
				recipeQueue = append(recipeQueue, recipeName)
			}
			continue
		}

		node := &RecipeTreeNode{Name: recipeName}
		var currentChildren []*RecipeTreeNode
		for _, r := range GetValidRecipe(recipeName, owned) {
			childTree := recipeToTree[r]
			if childTree != nil {
				currentChildren = append(currentChildren, childTree)
			}
		}
		SetChildren(node, [][]*RecipeTreeNode{currentChildren})
		canMake := GetCreatedBy(recipeName)
		for _, r := range canMake {
			if owned[r] {
				continue
			}
			recipeQueue = append(recipeQueue, r)
		}

		nodebfs := queue[0]
		queue = queue[1:]

		recipe, exists := recipeMap[nodebfs.Name]
		if !exists {
			fmt.Println("Recipe not found:", nodebfs.Name)
			continue
		}

		var children [][]*RecipeTreeNode

		for _, r := range recipe.Recipes {
			childNodes := []*RecipeTreeNode{}
			for _, name := range r {
				childNode := &RecipeTreeNode{Name: name}
				childNodes = append(childNodes, childNode)
				queue = append(queue, childNode)
				bfsVisited[name] = childNode
			}

			if len(r) == 2 && IsBaseElement(r[0]) && IsBaseElement(r[1]) {
				if CalculateTotalCompleteRecipes(root) >= maxRecipes {
					return
				}
			}

			children = append(children, childNodes)
		}
		SetChildren(nodebfs, children)
		*nodesVisited++
		synchronizeRecipeTree(bfsVisited, recipeToTree)
		if CalculateTotalCompleteRecipes(root) >= maxRecipes {
			return
		}
	}
}

// bidirection from the bottom, use as reference
func BuildFromBottom(
	recipeMap map[string]Recipe,
	recipeToTree map[string]*RecipeTreeNode,
	targetTier int,
) {
	owned := make(map[string]bool)
	tier := 0
	recipeQueue := []string{}
	for _, base := range GetAllElements(tier) {
		temp := &RecipeTreeNode{Name: base}
		recipeToTree[base] = temp
		recipeQueue = append(recipeQueue, base)
	}

	// Adjusted to use && to avoid infinite loop
	for len(recipeQueue) > 0 && !OwnAllTier(targetTier, owned) {
		recipeName := recipeQueue[0]
		recipeQueue = recipeQueue[1:]

		if owned[recipeName] {
			continue
		}

		if IsBaseElement(recipeName) {
			owned[recipeName] = true
			canMake := GetCreatedBy(recipeName)
			for _, r := range canMake {
				if owned[r] {
					continue
				}
				recipeQueue = append(recipeQueue, r)
			}
			continue
		}

		// Ensure no infinite loops for recipes that cannot be made
		if !CanMakeRecipe(recipeName, owned) {
			// To avoid re-adding the same recipe, we should check if it's already in the queue
			if !contains(recipeQueue, recipeName) {
				recipeQueue = append(recipeQueue, recipeName)
			}
			continue
		}

		owned[recipeName] = true

		node := &RecipeTreeNode{Name: recipeName}
		// Simplified children setup
		var currentChildren []*RecipeTreeNode
		for _, r := range GetValidRecipe(recipeName, owned) {
			childTree := recipeToTree[r]
			if childTree != nil {
				currentChildren = append(currentChildren, childTree)
			}
		}
		SetChildren(node, [][]*RecipeTreeNode{currentChildren})
		recipeToTree[recipeName] = node
		canMake := GetCreatedBy(recipeName)
		for _, r := range canMake {
			if owned[r] {
				continue
			}
			recipeQueue = append(recipeQueue, r)
		}
	}
}

func synchronizeRecipeTree(
	bfsVisited map[string]*RecipeTreeNode,
	recipeToTree map[string]*RecipeTreeNode,
) {
	for name, node := range bfsVisited {
		if existingNode, exists := recipeToTree[name]; exists {
			// Merge the children of the BFS node into the existing node
			SetChildren(existingNode, node.Children)
		} else {
			// If the node doesn't exist, add it to the recipeToTree, leave it as is
			continue
		}
	}
}

// Helper function to check if a recipe is in the queue
func contains(slice []string, item string) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}

func GetAllElements(tier int) []string {
	var elements []string
	for _, recipe := range RecipeMap {
		if recipe.Tier == tier {
			elements = append(elements, recipe.Name)
		}
	}
	return elements
}

func CanMakeRecipe(recipeName string, ownedMap map[string]bool) bool {
	recipe, exists := RecipeMap[recipeName]
	if !exists {
		return false
	}
	for _, ingredients := range recipe.Recipes {
		canMake := true
		for _, ingredient := range ingredients {
			if !ownedMap[ingredient] {
				canMake = false
				break
			}
		}
		if canMake {
			return true
		}
	}
	return false
}

func GetValidRecipe(recipeName string, ownedMap map[string]bool) []string {
	recipe, exists := RecipeMap[recipeName]
	if !exists {
		return nil
	}
	var validRecipes []string
	for _, ingredients := range recipe.Recipes {
		canMake := true
		for _, ingredient := range ingredients {
			if !ownedMap[ingredient] {
				canMake = false
				break
			}
		}
		if canMake {
			validRecipes = append(validRecipes, ingredients...)
			break
		}
	}
	return validRecipes
}

func OwnAllTier(tier int, ownedMap map[string]bool) bool {
	recipes := GetAllElements(tier)
	for _, recipe := range recipes {
		if !ownedMap[recipe] {
			return false
		}
	}
	return true
}

func GetCreatedBy(recipeName string) []string {
	canCreate := []string{}
	for _, recipe := range RecipeMap {
		for _, ingredients := range recipe.Recipes {
			for _, ingredient := range ingredients {
				if ingredient == recipeName {
					canCreate = append(canCreate, recipe.Name)
					break
				}
			}
		}
	}
	return canCreate
}
