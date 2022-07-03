package main

import (
	"fmt"
	"log"
	"sort"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Context struct {
	id            int64
	account       *Account
	subscriptions map[string]*Subscription
	caches        map[string]map[string]interface{}
}

func InitContexts() error {
	accounts, err := SharedFirebase().GetAccounts()
	if err != nil {
		return err
	}

	for _, account := range accounts {
		_, err := NewContext(account.Id, account.Kind)
		if err != nil {
			return err
		}
	}

	return nil
}

func NewContext(id int64, kind int) (*Context, error) {
	context := contexts[id]
	if context != nil {
		return context, nil
	}

	context = &Context{
		id:            id,
		subscriptions: make(map[string]*Subscription),
		caches:        make(map[string]map[string]interface{}),
	}

	account, err := SharedFirebase().GetAccount(id)
	if err != nil {
		return nil, err
	}
	if account == nil {
		account = &Account{
			Id:   id,
			Kind: kind,
		}
		err = SharedFirebase().SaveAccount(account)
		if err != nil {
			return nil, err
		}
	}
	context.account = account

	if subscriptions, err := SharedFirebase().GetSubscriptions(account); err != nil {
		return nil, err
	} else {
		for id, subscription := range subscriptions {
			context.subscriptions[id] = subscription

			cache, err := SharedFirebase().GetFeedCache(account, subscription)
			if err != nil {
				return nil, err
			}
			context.caches[id] = cache
		}
	}

	for _, subscription := range context.subscriptions {
		err = context.StartObserving(subscription)
		if err != nil {
			return nil, err
		}
	}

	contexts[account.Id] = context

	return context, nil
}

func (context *Context) StartObserving(subscription *Subscription) error {
	observer := &Observer{
		identifier: context.id,
		handler: func(items map[string]*Item) {
			if len(items) == 0 {
				return
			}

			old := make(map[string]interface{})
			new := make(map[string]interface{})

			for id, cache := range context.caches[subscription.Id] {
				if items[id] == nil {
					old[id] = cache
				}
			}

			for _, item := range items {
				if context.caches[subscription.Id][item.id] == nil {
					new[item.id] = map[string]interface{}{
						"pushed":    true,
						"timestamp": time.Now().Unix(),
					}

					msg := fmt.Sprintf("[%s](%s)", item.title, item.link)
					err := session.Send(context.id, msg)
					if err != nil {
						log.Println(err)
						return
					}
				}
			}

			if len(new) == 0 && len(old) == 0 {
				return
			}

			for id := range old {
				delete(context.caches[subscription.Id], id)
			}
			for id, cache := range new {
				context.caches[subscription.Id][id] = cache
			}
			SharedFirebase().SetFeedCache(context.account, subscription, context.caches[subscription.Id])
		},
	}
	SharedMonitor().AddObserver(observer, subscription.Link)

	return nil
}

func (context *Context) StopObserving(subscription *Subscription) error {
	SharedMonitor().RemoveObserver(context.id, subscription.Link)

	return nil
}

func (context *Context) Subscribe(channel *Channel) (*Subscription, error) {
	id := channel.id

	subscription := context.subscriptions[id]
	if subscription != nil {
		return nil, fmt.Errorf(`Subscription [%s](%s) exists`, subscription.Title, subscription.Link)
	}

	subscription = &Subscription{
		Id:        id,
		Link:      channel.link,
		Title:     channel.title,
		Timestamp: time.Now().Unix(),
	}
	context.subscriptions[id] = subscription

	context.caches[id] = make(map[string]interface{})

	err := fb.AddSubscription(context.account, subscription)
	if err != nil {
		return nil, err
	}

	return subscription, nil
}

func (context *Context) Unsubscribe(subscription *Subscription) error {
	err := fb.DeleteSubscription(context.account, subscription)
	if err != nil {
		return err
	}
	delete(context.subscriptions, subscription.Id)

	err = fb.DeleteFeedCache(context.account, subscription)
	if err != nil {
		return err
	}
	delete(context.caches, subscription.Id)

	return err
}

func (context *Context) SetItemsPushed(subscription *Subscription, items []*Item) error {
	for _, item := range items {
		context.caches[subscription.Id][item.id] = map[string]interface{}{
			"pushed":    true,
			"timestamp": time.Now().Unix(),
		}
	}

	return SharedFirebase().SetFeedCache(context.account, subscription, context.caches[subscription.Id])
}

func (context *Context) GetSubscriptions() []*Subscription {
	subscriptions := make([]*Subscription, 0)
	for _, subscription := range context.subscriptions {
		subscriptions = append(subscriptions, subscription)
	}

	sort.SliceStable(subscriptions, func(i, j int) bool {
		return subscriptions[i].Timestamp < subscriptions[j].Timestamp
	})

	return subscriptions
}
