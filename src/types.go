package main

type Account struct {
	Id     int64 `firestore:"id"`
	Kind   int   `firestore:"kind"`
	Status int   `firestore:"status"`
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
