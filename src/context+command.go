package main

import (
	"fmt"
	"strconv"
)

// Handlers

func (context *Context) HandleListCommand() string {
	subscriptions := context.GetSubscriptions()
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
	} else if subscription, err := context.Subscribe(channel); err != nil {
		return `Subscribe failed.`
	} else if err := context.SetItemsPushed(subscription, items); err != nil {
		return `Subscribe failed.`
	} else if err := context.StartObserving(subscription); err != nil {
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
	subscriptions := context.GetSubscriptions()

	index, err := strconv.Atoi(args)
	if err != nil || index <= 0 || index > len(subscriptions) {
		return fmt.Sprintf(`Invalid index.
			
%s`, context.HandleListCommand())
	}

	index -= 1

	subscription := subscriptions[index]

	if err := context.Unsubscribe(subscription); err != nil {
		return `Unsubscribe failed.`
	} else if err := context.StopObserving(subscription); err != nil {
		return `Unsubscribe failed.`
	} else {
		return fmt.Sprintf(`[%s](%s) unsubscribed.`, subscription.Title, subscription.Link)
	}
}

func (context *Context) HandleHotCommand(args string) string {
	if statistics, err := SharedFirebase().GetTopSubscriptions(5); err != nil {
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
