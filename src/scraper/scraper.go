package scraper

import (
	"encoding/json"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly"
)

type ElementWithRecipes struct {
	Element  string     `json:"element"`
	Tier     int        `json:"tier"` // <- fixed struct tag
	ImageURL string     `json:"image_url"`
	Recipes  [][]string `json:"recipes"`
}

func ExcludedElements() map[string]bool {
	excluded := make(map[string]bool)
	c := colly.NewCollector()
	c.OnHTML("li.category-page__member", func(e *colly.HTMLElement) {
		name := strings.TrimSpace(e.ChildText("a.category-page__member-link"))
		if name != "" {
			excluded[name] = true
		}
	})
	if err := c.Visit("https://little-alchemy.fandom.com/wiki/Category:Myths_and_Monsters"); err != nil {
		log.Fatal(err)
	}
	excluded["Time"] = true
	return excluded
}

func FindRecipes() {
	excluded := ExcludedElements()
	baseEls := map[string]bool{"Fire": true, "Earth": true, "Water": true, "Air": true}

	var elements []ElementWithRecipes

	c := colly.NewCollector()

	c.OnHTML("h3", func(e *colly.HTMLElement) {
		headline := e.ChildText("span.mw-headline")


		if headline == "Starting elements" {
			tableSel := e.DOM.NextUntil("h3").FilterFunction(func(_ int, s *goquery.Selection) bool {
				return s.Is("table.list-table.col-list.icon-hover")
			}).First()

			tableSel.Find("tbody tr").Each(func(_ int, row *goquery.Selection) {
				name := strings.TrimSpace(row.Find("td:nth-of-type(1) a").Text())
				if name == "" || excluded[name] || !baseEls[name] {
					return
				}
				imgURL, _ := row.Find("td:nth-of-type(1) a").Attr("href")
				if imgURL == "" {
					imgURL = "No image"
				}

				elements = append(elements, ElementWithRecipes{
					Element:  name,
					Tier:     0,
					ImageURL: imgURL,
					Recipes:  [][]string{{}}, // base element has no recipes
				})
			})
			return
		}

		if !strings.HasPrefix(headline, "Tier ") {
			return
		}
		parts := strings.Fields(headline)
		tier, err := strconv.Atoi(parts[1])
		if err != nil {
			return
		}

		tableSel := e.DOM.NextUntil("h3").FilterFunction(func(_ int, s *goquery.Selection) bool {
			return s.Is("table.list-table.col-list.icon-hover")
		}).First()

		tableSel.Find("tbody tr").Each(func(_ int, row *goquery.Selection) {
			name := strings.TrimSpace(row.Find("td:nth-of-type(1) a").Text())
			if name == "" || excluded[name] || baseEls[name] {
				return
			}
			imgURL, _ := row.Find("td:nth-of-type(1) a").Attr("href")
			if imgURL == "" {
				imgURL = "No image"
			}

			var recipes [][]string
			row.Find("td:nth-of-type(2) li").Each(func(_ int, li *goquery.Selection) {
				var comps []string
				li.Find("a").Each(func(_ int, a *goquery.Selection) {
					t := strings.TrimSpace(a.Text())
					if t != "" && !excluded[t] {
						comps = append(comps, t)
					}
				})
				if len(comps) == 2 {
					recipes = append(recipes, comps)
				}
			})

			elements = append(elements, ElementWithRecipes{
				Element:  name,
				Tier:     tier,
				ImageURL: imgURL,
				Recipes:  recipes,
			})
		})
	})


	if err := c.Visit("https://little-alchemy.fandom.com/wiki/Elements_(Little_Alchemy_2)"); err != nil {
		log.Fatal(err)
	}


	out, err := json.MarshalIndent(elements, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile("recipes.json", out, 0644); err != nil {
		log.Fatal("Failed to write recipes.json:", err)
	}
}
