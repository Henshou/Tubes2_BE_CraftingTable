package scraper

import (
	"strings"

	"github.com/gocolly/colly"
)

func FindRecipes(element string) [][]string {
	var recipes [][]string

	c := colly.NewCollector()

	c.OnHTML("tr", func(e *colly.HTMLElement) {
		result := e.ChildText("td:nth-of-type(1) a")
		if strings.EqualFold(result, element) {
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
		}
	})

	c.Visit("https://little-alchemy.fandom.com/wiki/Elements_(Little_Alchemy_2)")
	return recipes
}
