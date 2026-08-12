// Package content holds the blog and principles data. Articles are loaded
// from data/articles.json (see LoadArticles); principles live in the generated
// data_principles.go. This file defines the shapes and the accessor functions.
package content

import (
	"encoding/json"
	"os"
)

// articles holds the loaded blog posts. Empty until LoadArticles runs at
// startup; see also the generated principles in data_principles.go.
var articles []Article

// Article is a single blog post.
type Article struct {
	Slug         string   `json:"slug"`
	Title        string   `json:"title"`
	Subtitle     string   `json:"subtitle"`
	Date         string   `json:"date"`
	Tags         []string `json:"tags"`
	Excerpt      string   `json:"excerpt"`
	Image        string   `json:"image"`
	ImageCaption string   `json:"imageCaption"`
	InitialLove  int      `json:"initialLove"`
	Star         bool     `json:"star"`
	Featured     bool     `json:"featured"`
	Body         string   `json:"body"`
}

// Principle is one numbered maxim from the home page.
type Principle struct {
	N     int
	Title string
	Body  string
}

// Articles returns every imported article.
func Articles() []Article {
	return articles
}

// LoadArticles replaces the in-memory articles with the contents of the JSON
// file at path. It is called once at startup so the site renders from the same
// file the client-side command palette reads.
func LoadArticles(path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var arts []Article
	if err := json.Unmarshal(b, &arts); err != nil {
		return err
	}
	articles = arts
	return nil
}

// FindArticle returns the article with the given slug, or nil.
func FindArticle(slug string) *Article {
	for i := range articles {
		if articles[i].Slug == slug {
			return &articles[i]
		}
	}
	return nil
}

// Principles returns every imported principle.
func Principles() []Principle {
	return principles
}
