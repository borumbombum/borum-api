// Package content holds the blog and principles data imported from the
// doags static-port. The data itself lives in the generated data_*.go files;
// this file defines the shapes and the accessor functions.
package content

// Article is a single blog post.
type Article struct {
	Slug         string
	Title        string
	Subtitle     string
	Date         string
	Tags         []string
	Excerpt      string
	Image        string
	ImageCaption string
	InitialLove  int
	Star         bool
	Featured     bool
	Body         string
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
