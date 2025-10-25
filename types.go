package main

import "time"

const (
	errTelegramBotTokenNotFound = "telegram bot token not found"

	errFirebaseCredentialNotFound = "firebase credential not found"
	errFirebaseDatabaseNotFound   = "firebase database not found"
)

type BotData struct {
}

type FeedStatus struct {
	PublishedItems      map[string]bool `firestore:"published-items"`
	LatestPublishedTime time.Time `firestore:"latest-published-time"`
}

type UserData struct {
	Feeds      map[string]*Feed `firestore:"feeds"`
	FeedStatus map[string]*FeedStatus `firestore:"feed-status"`
}

type Monitor struct {
	observers map[string]map[int64]*Observer
	ticker    *time.Ticker
	quit      chan bool
}

type Observer struct {
	identifier int64
	handler    func(items []*Item)
}

type Feed struct {
	Id             string    `firestore:"id"`
	Link           string    `firestore:"link"`
	Title          string    `firestore:"title"`
	Topic          int       `firestore:"topic"`
	SubscribedTime time.Time `firestore:"subscribed-time"`
}

type Item struct {
	Id            string
	Title         string
	Link          string
	PublishedTime time.Time
}

const (
	CmdList     = "list"
	CmdAdd      = "add"
	CmdDelete   = "delete"
	CmdTop      = "top"
	CmdSetTopic = "settopic"
)
