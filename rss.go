package main

import (
	"crypto/md5"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/mmcdole/gofeed"
)

func fetchFeed(url string) (*Feed, []*Item, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_4) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/83.0.4103.97 Safari/537.36")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, nil, err
	}

	parser := gofeed.NewParser()
	feed, err := parser.Parse(resp.Body)
	if err != nil {
		return nil, nil, err
	}

	var items []*Item
	for _, item := range feed.Items {
		items = append(items, &Item{
			Id:            getItemID(item),
			Title:         item.Title,
			Link:          item.Link,
			PublishedTime: *item.PublishedParsed,
		})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].PublishedTime.Before(items[j].PublishedTime)
	})

	return &Feed{
		Id:             getFeedID(feed),
		Link:           url,
		Title:          feed.Title,
		SubscribedTime: time.Now(),
	}, items, nil
}

func getFeedID(feed *gofeed.Feed) string {
	var param string
	if feed.Link != "" {
		param = feed.Link
	} else {
		return ""
	}
	return fmt.Sprintf("%x", md5.Sum([]byte(param)))
}

func getItemID(item *gofeed.Item) string {
	var param string
	if item.GUID != "" {
		param = item.GUID
	} else if item.Link != "" {
		param = item.Link
	} else {
		return ""
	}
	return fmt.Sprintf("%x", md5.Sum([]byte(param)))
}
