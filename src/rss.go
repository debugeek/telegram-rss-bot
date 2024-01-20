package main

import (
	"crypto/md5"
	"fmt"
	"net/http"
	"sort"

	"github.com/mmcdole/gofeed"
)

func fetchItems(url string) (*Channel, []*Item, error) {
	feed, err := fetchFeed(url)
	if err != nil {
		return nil, nil, err
	}

	channel := &Channel{
		id:          fmt.Sprintf("%x", md5.Sum([]byte(feed.Link))),
		title:       feed.Title,
		description: feed.Description,
		link:        url,
	}

	var items []*Item
	for _, item := range feed.Items {
		items = append(items, &Item{
			id:    fmt.Sprintf("%x", md5.Sum([]byte(item.GUID))),
			title: item.Title,
			link:  item.Link,
			date:  item.Published,
		})
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].date < items[j].date
	})

	return channel, items, nil
}

func fetchFeed(url string) (*gofeed.Feed, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_4) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/83.0.4103.97 Safari/537.36")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	parser := gofeed.NewParser()
	feed, err := parser.Parse(resp.Body)

	return feed, err
}
