package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/PuerkitoBio/goquery"
)

type OGTags struct {
	Title       string
	Description string
	Image       string
	URL         string
}

func fetchTags(url string) (OGTags, error) {
	res, err := http.Get(url)
	if err != nil {
		return OGTags{}, fmt.Errorf("fetch error: invalid url: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return OGTags{}, fmt.Errorf("fetch error: invalid response: %v", err)
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)

	if err != nil {
		return OGTags{}, fmt.Errorf("fetch error: can't read url: %v", err)
	}

	var tags OGTags

	doc.Find("meta[property='og:title']").Each(func(index int, item *goquery.Selection) {
		if title, exists := item.Attr("content"); exists {
			tags.Title = title
		}
	})

	doc.Find("meta[property='og:description']").Each(func(index int, item *goquery.Selection) {
		if desc, exists := item.Attr("content"); exists {
			tags.Description = desc
		}
	})

	doc.Find("meta[property='og:image']").Each(func(index int, item *goquery.Selection) {
		if img, exists := item.Attr("content"); exists {
			tags.Image = img
		}
	})

	doc.Find("meta[property='og:url']").Each(func(index int, item *goquery.Selection) {
		if url, exists := item.Attr("content"); exists {
			tags.Title = url
		}
	})

	return tags, nil
}

func main() {
	http.HandleFunc("/get_tags", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		query := r.URL.Query()
		url := query.Get("url")

		if url == "" {
			http.Error(w, "Missing 'url' parameter", http.StatusBadRequest)
			return
		}

		tags, err := fetchTags(url)
		if err != nil {
			http.Error(w, "Can't fetch tags", http.StatusBadRequest)
		}

		w.Header().Set("Content-Type", "application/json")

		if err := json.NewEncoder(w).Encode(tags); err != nil {
			http.Error(w, fmt.Sprintf("Error encoding JSON: %v", err), http.StatusInternalServerError)
			return
		}
	})

	log.Println("Server is running.")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
