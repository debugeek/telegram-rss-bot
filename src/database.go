package main

import "log"

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
			db = &MemoryDatabase{}
		}
		db.InitDatabase()
	})
}

type MemoryDatabase struct{}

func (db *MemoryDatabase) InitDatabase() {
	log.Println(`Memory cache initialized`)
}
func (db *MemoryDatabase) GetAccounts() ([]*Account, error) {
	return nil, nil
}
func (db *MemoryDatabase) GetAccount(id int64) (*Account, error) {
	return nil, nil
}
func (db *MemoryDatabase) SaveAccount(account *Account) error {
	return nil
}
func (db *MemoryDatabase) GetSubscriptions(account *Account) (map[string]*Subscription, error) {
	return nil, nil
}
func (db *MemoryDatabase) AddSubscription(account *Account, subscription *Subscription) error {
	return nil
}
func (db *MemoryDatabase) DeleteSubscription(account *Account, subscription *Subscription) error {
	return nil
}
func (db *MemoryDatabase) GetFeedCache(account *Account, subscription *Subscription) (map[string]interface{}, error) {
	return nil, nil
}
func (db *MemoryDatabase) SetFeedCache(account *Account, subscription *Subscription, cache map[string]interface{}) error {
	return nil
}
func (db *MemoryDatabase) DeleteFeedCache(account *Account, subscription *Subscription) error {

	return nil
}
func (db *MemoryDatabase) GetTopSubscriptions(num int) ([]*SubscriptionStatistic, error) {
	return nil, nil
}
