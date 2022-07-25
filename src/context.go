package main

import (
	"fmt"
	"log"
	"sort"
	"strconv"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Context struct {
	id            int64
	account       *Account
	subscriptions map[string]*Subscription
	caches        map[string]map[string]interface{}
}

func InitContext() {
	accounts, err := fb.GetAccounts()
	if err != nil {
		log.Println(err)
		return
	}

	for _, account := range accounts {
		context := &Context{
			id:            account.Id,
			account:       account,
			subscriptions: make(map[string]*Subscription),
			caches:        make(map[string]map[string]interface{}),
		}

		if subscriptions, err := fb.GetSubscriptions(account); err != nil {
			log.Println(err)
			continue
		} else {
			for id, subscription := range subscriptions {
				context.subscriptions[id] = subscription

				cache, err := fb.GetFeedCache(account, subscription)
				if err != nil {
					log.Println(err)
					continue
				}
				context.caches[id] = cache
			}
		}

		contexts[account.Id] = context

		for _, subscription := range context.subscriptions {
			context.startObserving(subscription)
		}
	}
}

func GetCachedContext(id int64, kind int) *Context {
	return contexts[id]
}

func CacheContext(context *Context) {
	contexts[context.id] = context
}

func (context *Context) startObserving(subscription *Subscription) error {
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
					err := session.Send(context, msg)
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
			fb.SetFeedCache(context.account, subscription, context.caches[subscription.Id])
		},
	}
	monitor.AddObserver(observer, subscription.Link)

	return nil
}

func (context *Context) stopObserving(subscription *Subscription) error {
	monitor.RemoveObserver(context.id, subscription.Link)
	return nil
}

func (context *Context) subscribe(channel *Channel) (*Subscription, error) {
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

func (context *Context) unsubscribe(subscription *Subscription) error {
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

func (context *Context) setItemsHavePushed(subscription *Subscription, items []*Item) error {
	for _, item := range items {
		context.caches[subscription.Id][item.id] = map[string]interface{}{
			"pushed":    true,
			"timestamp": time.Now().Unix(),
		}
	}

	return fb.SetFeedCache(context.account, subscription, context.caches[subscription.Id])
}

func (context *Context) getSubscriptions() []*Subscription {
	subscriptions := make([]*Subscription, 0)
	for _, subscription := range context.subscriptions {
		subscriptions = append(subscriptions, subscription)
	}

	sort.SliceStable(subscriptions, func(i, j int) bool {
		return subscriptions[i].Timestamp < subscriptions[j].Timestamp
	})

	return subscriptions
}

// Command Handler

func (context *Context) HandleListCommand() string {
	subscriptions := context.getSubscriptions()
	if len(subscriptions) == 0 {
		return `Your list is empty.`
	}

	var message string
	for idx, subscription := range subscriptions {
		message += fmt.Sprintf("%d. [%s](%s) \n", idx+1, subscription.Title, subscription.Link)
	}
	return message
}

func (context *Context) HandleSubscribeCommand(args string) string {
	if len(args) == 0 || !isValidURL(args) {
		return `Unable to parse the url.`
	}

	if channel, items, err := FetchChannel(args); err != nil {
		return `Fetch error.`
	} else if subscription, err := context.subscribe(channel); err != nil {
		return `Subscribe failed.`
	} else if err := context.setItemsHavePushed(subscription, items); err != nil {
		return `Subscribe failed.`
	} else if err := context.startObserving(subscription); err != nil {
		return `Subscribe failed.`
	} else {
		if len(items) == 0 {
			return fmt.Sprintf(`[%s](%s) subscribed.`, subscription.Title, subscription.Link)
		} else {
			return fmt.Sprintf(`[%s](%s) subscribed. Here is the latest feed from the channel.
			
[%s](%s)`, subscription.Title, subscription.Link, items[0].title, items[0].link)
		}
	}
}

func (context *Context) HandleUnsubscribeCommand(args string) string {
	subscriptions := context.getSubscriptions()

	index, err := strconv.Atoi(args)
	if err != nil || index <= 0 || index > len(subscriptions) {
		return fmt.Sprintf(`Invalid index.
			
%s`, context.HandleListCommand())
	}

	index -= 1

	subscription := subscriptions[index]

	if err := context.unsubscribe(subscription); err != nil {
		return `Unsubscribe failed.`
	} else if err := context.stopObserving(subscription); err != nil {
		return `Unsubscribe failed.`
	} else {
		return fmt.Sprintf(`[%s](%s) unsubscribed.`, subscription.Title, subscription.Link)
	}
}

func (context *Context) HandleHotCommand(args string) string {
	if statistics, err := fb.GetTopSubscriptions(5); err != nil {
		return `Oops, something wrong happened.`
	} else if len(statistics) == 0 {
		return "Not enough data."
	} else {
		var message string
		for idx, statistic := range statistics {
			message += fmt.Sprintf("%d. [%s](%s) (👥 %d)\n", idx+1, statistic.Subscription.Title, statistic.Subscription.Link, statistic.Count)
		}
		return message
	}

}
