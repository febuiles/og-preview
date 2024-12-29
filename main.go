package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/redis/go-redis/v9"
)

type OGTags struct {
	Title       string
	Description string
	Image       string
	URL         string
}

var rdb *redis.Client

func initRedis() {
	rdb = redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
}

// TODO tag parsing/creation is very brittle
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
			tags.URL = url
		}
	})

	return tags, nil
}

func main() {
	initRedis()
	http.HandleFunc("/get_tags", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		query := r.URL.Query()
		url := query.Get("url")

		if url == "" {
			http.Error(w, "Missing 'url' parameter", http.StatusBadRequest)
			return
		}

		ctx := context.Background()
		cacheKey := url

		cached, err := rdb.Get(ctx, cacheKey).Result()
		if err == redis.Nil {
			tags, err := fetchTags(url)
			if err != nil {
				http.Error(w, "Can't fetch tags", http.StatusBadRequest)
			}

			jsonData, err := json.Marshal(tags)

			if err != nil {
				http.Error(w, fmt.Sprintf("Error encoding JSON: %v", err), http.StatusInternalServerError)
				return
			}

			if err := rdb.Set(ctx, cacheKey, jsonData, 10*time.Minute).Err(); err != nil {
				log.Printf("Error caching to Redis: %v", err)
			}

			w.Write(jsonData)
		} else if err != nil {
			log.Printf("Redis error: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		} else {
			w.Write([]byte(cached))
		}
	})

	log.Println("Server is running.")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
