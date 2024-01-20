package main

type DatabaseProtocol interface {
	Reload()

	GetAccounts() ([]*Account, error)
	GetAccount(id int64) (*Account, error)
	SaveAccount(account *Account) error

	GetSubscriptions(account *Account) (map[string]*Subscription, error)
	AddSubscription(account *Account, subscription *Subscription) error
	RemoveSubscription(account *Account, subscription *Subscription) error

	GetRecentlyPublishedFeeds(account *Account, subscription *Subscription) (map[string]interface{}, error)
	SetRecentlyPublishedFeeds(account *Account, subscription *Subscription, cache map[string]interface{}) error
	ClearRecentlyPublishedFeeds(account *Account, subscription *Subscription) error

	GetTopSubscriptions(num int) ([]*SubscriptionStatistic, error)
}
