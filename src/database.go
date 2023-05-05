package main

type DatabaseProtocol interface {
	InitDatabase()
	GetAccounts() ([]*Account, error)
	GetAccount(id int64) (*Account, error)
	SaveAccount(account *Account) error
	GetSubscriptions(account *Account) (map[string]*Subscription, error)
	AddSubscription(account *Account, subscription *Subscription) error
	DeleteSubscription(account *Account, subscription *Subscription) error
	GetFeedCache(account *Account, subscription *Subscription) (map[string]interface{}, error)
	SetFeedCache(account *Account, subscription *Subscription, cache map[string]interface{}) error
	DeleteFeedCache(account *Account, subscription *Subscription) error
	GetTopSubscriptions(num int) ([]*SubscriptionStatistic, error)
}

func InitDatabase() {
	dbOnce.Do(func() {
		if len(args.FirebaseConf) != 0 || len(args.FirebaseConfEnvKey) != 0 {
			db = &Firebase{}
		} else {
			db = &MemCache{}
		}
		db.InitDatabase()
	})
}
