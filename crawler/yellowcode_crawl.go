package crawler

import (
	"context"
	"devread/custom_error"
	"devread/helper"
	"devread/model"
	"devread/repository"
	"fmt"
	"github.com/gocolly/colly/v2"
	"log"
	"runtime"
	"strings"
)

func YellowcodePost(postRepo repository.PostRepo) {
	c := colly.NewCollector()

	posts := []model.Post{}
	var yellowcodePost model.Post
	c.OnHTML("header[class=entry-header]", func(e *colly.HTMLElement) {
		yellowcodePost.Name = e.ChildText("h2.entry-title > a")
		yellowcodePost.Link = e.ChildAttr("h2.entry-title > a", "href")
		yellowcodePost.Tag = strings.ToLower(strings.Replace(
			strings.Replace(
				strings.Replace(
					e.ChildText("span.meta-category > a"), "\n", "", -1), "/", "", -1), "-", "", -1))
		yellowcodePost.PostID = helper.Hash(yellowcodePost.Name, yellowcodePost.Link)
		posts = append(posts, yellowcodePost)
	})

	c.OnScraped(func(r *colly.Response) {
		queue := helper.NewJobQueue(runtime.NumCPU())
		queue.Start()
		defer queue.Stop()

		for _, post := range posts {
			queue.Submit(&YellowcodeProcess{
				post:     post,
				postRepo: postRepo,
			})
		}
	})

	c.OnError(func(r *colly.Response, err error) {
		log.Println("Request URL:", r.Request.URL, "failed with response:", r, "\nError:", err)
	})

	listURL := []string{}
	for numb := 1; numb < 7; numb++ {
		trend := fmt.Sprintf("https://yellowcodebooks.com/category/lap-trinh-android/page/%d", numb)
		listURL = append(listURL, trend)
	}
	for numb := 1; numb < 6; numb++ {
		newest := fmt.Sprintf("https://yellowcodebooks.com/category/lap-trinh-java/page/%d", numb)
		listURL = append(listURL, newest)
	}

	for _, url := range listURL {
		c.Visit(url)
		fmt.Println(url)
	}
}

type YellowcodeProcess struct {
	post     model.Post
	postRepo repository.PostRepo
}

func (process *YellowcodeProcess) Process() {
	// select post by id
	cacheRepo, err := process.postRepo.SelectById(context.Background(), process.post.PostID)
	if err == custom_error.PostNotFound {
		// insert post to database
		fmt.Println("Add: ", process.post.Name)
		_, err = process.postRepo.Save(context.Background(), process.post)
		if err != nil {
			log.Println(err)
		}
		return
	}

	// update post
	if process.post.PostID != cacheRepo.PostID {
		fmt.Println("Updated: ", process.post.Name)
		_, err = process.postRepo.Update(context.Background(), process.post)
		if err != nil {
			log.Println(err)
		}
	}
}
