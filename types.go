package main

import "time"

const (
	errTelegramBotTokenNotFound = "telegram bot token not found"

	errFirebaseCredentialNotFound = "firebase credential not found"
	errFirebaseDatabaseNotFound   = "firebase database not found"
)

type BotData struct {
}

type UserData struct {
	subscriptions  map[string]*Subscription
	publishedFeeds map[string]map[string]interface{}
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

type Subscription struct {
	Id        string `firestore:"id"`
	Link      string `firestore:"link"`
	Title     string `firestore:"title"`
	Timestamp int64  `firestore:"timestamp"`
}

type Channel struct {
	id          string
	title       string
	description string
	link        string
}

type Item struct {
	id    string
	title string
	link  string
	date  string
}

type SubscriptionStatistic struct {
	Count        int64         `firestore:"count"`
	Subscription *Subscription `firestore:"subscription"`
}

const (
	CmdList   = "list"
	CmdAdd    = "add"
	CmdDelete = "delete"
	CmdTop    = "top"
)
