package main

type MemCache struct{}

func (c *MemCache) Reload() {}
func (c *MemCache) GetAccounts() ([]*Account, error) {
	return nil, nil
}
func (c *MemCache) GetAccount(id int64) (*Account, error) {
	return nil, nil
}
func (c *MemCache) SaveAccount(account *Account) error {
	return nil
}
func (c *MemCache) GetSubscriptions(account *Account) (map[string]*Subscription, error) {
	return nil, nil
}
func (c *MemCache) AddSubscription(account *Account, subscription *Subscription) error {
	return nil
}
func (c *MemCache) RemoveSubscription(account *Account, subscription *Subscription) error {
	return nil
}
func (c *MemCache) GetRecentlyPublishedFeeds(account *Account, subscription *Subscription) (map[string]interface{}, error) {
	return nil, nil
}
func (c *MemCache) SetRecentlyPublishedFeeds(account *Account, subscription *Subscription, cache map[string]interface{}) error {
	return nil
}
func (c *MemCache) ClearRecentlyPublishedFeeds(account *Account, subscription *Subscription) error {

	return nil
}
func (c *MemCache) GetTopSubscriptions(num int) ([]*SubscriptionStatistic, error) {
	return nil, nil
}
