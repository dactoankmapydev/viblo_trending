package helper

import (
	"context"
	"fmt"
	"github.com/gocolly/colly/v2"
	"github.com/labstack/gommon/log"
	"runtime"
	"strings"
	"tech_posts_trending/custom_error"
	"tech_posts_trending/model"
	"tech_posts_trending/repository"
	"time"
)

func ToidicodedaoPost(postRepo repository.PostRepo) {
	c := colly.NewCollector()
	c.SetRequestTimeout(30 * time.Second)

	posts := []model.Post{}
	var toidicodedaoPost model.Post

	c.OnHTML("footer[class=entry-meta]", func(e *colly.HTMLElement) {
		if toidicodedaoPost.Name == "" || toidicodedaoPost.Link == "" {
			return
		}
		toidicodedaoPost.Tags = strings.ToLower(e.ChildText("span.tag-links > a:last-child"))
		posts = append(posts, toidicodedaoPost)
	})

	c.OnHTML(".site-content .entry-title", func(e *colly.HTMLElement) {
		toidicodedaoPost.Name = e.Text
		toidicodedaoPost.Link = e.ChildAttr("h1.entry-title > a", "href")
		c.Visit(e.Request.AbsoluteURL(toidicodedaoPost.Link))
		if toidicodedaoPost.Name == "" || toidicodedaoPost.Link == "" {
			return
		}
		posts = append(posts, toidicodedaoPost)
	})

	c.OnScraped(func(r *colly.Response) {
		queue := NewJobQueue(runtime.NumCPU())
		queue.Start()
		defer queue.Stop()

		for _, post := range posts {
			queue.Submit(&ToidicodedaoProcess{
				post:     post,
				postRepo: postRepo,
			})
		}
	})

	c.OnError(func(r *colly.Response, err error) {
		log.Error("Request URL:", r.Request.URL, "failed with response:", r, "\nError:", err)
	})

	for i := 1; i < 32; i++ {
		fullURL := fmt.Sprintf("https://toidicodedao.com/category/chuyen-coding/page/%d", i)
		c.Visit(fullURL)
		fmt.Println(fullURL)
	}
}

type ToidicodedaoProcess struct {
	post     model.Post
	postRepo repository.PostRepo
}

func (process *ToidicodedaoProcess) Process() {
	// select post by name
	cacheRepo, err := process.postRepo.SelectPostByName(context.Background(), process.post.Name)
	if err == custom_error.PostNotFound {
		// insert post to database
		_, err = process.postRepo.SavePost(context.Background(), process.post)
		if err != nil {
			log.Error(err)
		}
		return
	}

	// update post
	if process.post.Name != cacheRepo.Name {
		log.Info("Updated: ", process.post.Name)
		_, err = process.postRepo.UpdatePost(context.Background(), process.post)
		if err != nil {
			log.Error(err)
		}
	}
}