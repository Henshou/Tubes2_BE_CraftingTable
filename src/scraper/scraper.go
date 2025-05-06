package scraper

import (
	"encoding/json"
	"log"
	"os"
	"strings"

	"github.com/gocolly/colly"
)

type ElementWithRecipes struct {
	Element string     `json:"element"`
	Recipes [][]string `json:"recipes"`
}

func FindRecipes() {
	var elements []ElementWithRecipes

	baseElements := map[string]bool{
		"Fire":  true,
		"Earth": true,
		"Water": true,
		"Air":   true,
	}

	c := colly.NewCollector()

	c.OnHTML("tr", func(e *colly.HTMLElement) {
		elementName := e.ChildText("td:nth-of-type(1) a")
		if elementName == "" {
			return
		}

		var recipes [][]string
		e.ForEach("td:nth-of-type(2) li", func(_ int, li *colly.HTMLElement) {
			var components []string
			li.ForEach("a", func(_ int, a *colly.HTMLElement) {
				text := strings.TrimSpace(a.Text)
				if text != "" {
					components = append(components, text)
				}
			})
			if len(components) == 2 {
				recipes = append(recipes, components)
			}
		})

		if len(recipes) > 0 && !baseElements[elementName] {
			elements = append(elements, ElementWithRecipes{
				Element: elementName,
				Recipes: recipes,
			})
		} else if baseElements[elementName] {
			elements = append(elements, ElementWithRecipes{
				Element: elementName,
				Recipes: [][]string{{}},
			})
		}
	})

	err := c.Visit("https://little-alchemy.fandom.com/wiki/Elements_(Little_Alchemy_2)")
	if err != nil {
		log.Fatal(err)
	}

	jsonData, err := json.MarshalIndent(elements, "", "  ")
	if err != nil {
		log.Fatal(err)
	}

	err = os.WriteFile("recipes.json", jsonData, 0644)
	if err != nil {
		log.Fatal("Failed to write JSON to file:", err)
	}
}
