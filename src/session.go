package main

import (
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

type Session struct {
	bot      *tgbotapi.BotAPI
	monitor  *Monitor
	contexts map[int64]*Context
	db       DatabaseProtocol
}

func NewSession(token string, db DatabaseProtocol) *Session {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Fatal(err)
	}

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 10
	_, err = bot.GetUpdates(u)
	if err != nil {
		log.Fatal(err)
	}

	return &Session{
		bot: bot,
		monitor: &Monitor{
			observers: make(map[string]map[int64]*Observer),
		},
		contexts: make(map[int64]*Context),
		db:       db,
	}
}

func (session *Session) reload() {
	accounts, err := session.db.GetAccounts()
	if err != nil {
		log.Fatal(err)
	}

	for _, account := range accounts {
		context := &Context{
			id:             account.Id,
			account:        account,
			subscriptions:  make(map[string]*Subscription),
			publishedFeeds: make(map[string]map[string]interface{}),
		}

		if subscriptions, err := session.db.GetSubscriptions(account); err != nil {
			log.Println(err)
			continue
		} else {
			for id, subscription := range subscriptions {
				context.subscriptions[id] = subscription

				publishedFeeds, err := session.db.GetRecentlyPublishedFeeds(account, subscription)
				if err != nil {
					log.Println(err)
					continue
				}
				context.publishedFeeds[id] = publishedFeeds
			}
		}

		for _, subscription := range context.subscriptions {
			session.startObserving(context, subscription)
		}

		session.contexts[account.Id] = context
	}
}

func (session *Session) startObserving(context *Context, subscription *Subscription) error {
	observer := &Observer{
		identifier: context.id,
		handler: func(items []*Item) {
			session.processFeedItems(context, items, subscription)
		},
	}
	session.monitor.addObserver(observer, subscription.Link)
	return nil
}

func (session *Session) stopObserving(context *Context, subscription *Subscription) error {
	session.monitor.removeObserver(context.id, subscription.Link)
	return nil
}

func (session *Session) getContext(id int64, kind int) *Context {
	return session.contexts[id]
}

func (session *Session) setContext(context *Context) {
	session.contexts[context.id] = context
}

func (session *Session) run() {
	go session.monitor.runLoop()
	go session.runLoop()
}

func (session *Session) runLoop() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 10

	updates, err := session.bot.GetUpdatesChan(u)
	if err != nil {
		log.Println(err)
		return
	}

	time.Sleep(time.Millisecond * 500)
	updates.Clear()

	for update := range updates {
		var message *tgbotapi.Message
		if update.Message != nil {
			message = update.Message
		} else if update.ChannelPost != nil {
			message = update.ChannelPost
		}
		if message == nil {
			log.Println("unabled to handle update")
			continue
		}

		log.Println(message.Text)

		id := message.Chat.ID
		kind := -1
		if message.Chat.IsPrivate() {
			kind = 0
		} else if message.Chat.IsGroup() || message.Chat.IsSuperGroup() {
			kind = 1
		} else if message.Chat.IsChannel() {
			kind = 2
		}

		context := session.getContext(id, kind)
		if context == nil {
			account := &Account{
				Id:     id,
				Kind:   kind,
				Status: 1,
			}
			err = session.db.SaveAccount(account)
			if err != nil {
				log.Println(err)
				continue
			}
			context = &Context{
				id:             id,
				account:        account,
				subscriptions:  make(map[string]*Subscription),
				publishedFeeds: make(map[string]map[string]interface{}),
			}
			session.setContext(context)
		}

		session.processMessage(context, message)
	}
}

// Command Processors

func (session *Session) processMessage(context *Context, message *tgbotapi.Message) {
	if context.account.Status == -1 {
		context.account.Status = 1
		session.db.SaveAccount(context.account)
	}

	if message.IsCommand() {
		switch strings.ToLower(message.Command()) {
		case "start":
			{
				session.send(context, "Greetings.", false)
				break
			}

		case "list":
			{
				response := session.processListCommand(context)
				session.reply(context, message.MessageID, response, false)
				break
			}

		case "add", "subscribe", "sub":
			{
				args := message.CommandArguments()
				response := session.processSubscribeCommand(context, args)
				session.reply(context, message.MessageID, response, false)
				break
			}

		case "delete", "del", "unsubscribe", "unsub":
			{
				args := message.CommandArguments()
				response := session.processUnsubscribeCommand(context, args)
				session.reply(context, message.MessageID, response, false)
				break
			}

		case "hot", "top":
			{
				args := message.CommandArguments()
				response := session.processTopSubscriptionsCommand(context, args)
				session.reply(context, message.MessageID, response, false)
				break
			}
		default:
			break
		}
	}
}

func (session *Session) processListCommand(context *Context) string {
	subscriptions := session.getSubscriptions(context)
	if len(subscriptions) == 0 {
		return `Your list is empty.`
	}

	var message string
	for idx, subscription := range subscriptions {
		message += fmt.Sprintf("%d. %s \n", idx+1, HTMLLink(subscription.Title, subscription.Link))
	}
	return message
}

func (session *Session) processSubscribeCommand(context *Context, args string) string {
	if len(args) == 0 || !isValidURL(args) {
		return `Unable to parse the url.`
	}

	if channel, items, err := fetchItems(args); err != nil {
		return err.Error()
	} else if subscription, err := session.subscribe(context, channel); err != nil {
		return err.Error()
	} else if err := session.updateRecentlyPublishedFeeds(context, subscription, items); err != nil {
		return err.Error()
	} else if err := session.startObserving(context, subscription); err != nil {
		return err.Error()
	} else {
		if len(items) == 0 {
			return fmt.Sprintf(`%s subscribed.`, HTMLLink(subscription.Title, subscription.Link))
		} else {
			latestItem := items[len(items)-1]
			return fmt.Sprintf(`%s subscribed. Here is the latest feed from the channel.
            
%s`, HTMLLink(subscription.Title, subscription.Link), HTMLLink(latestItem.title, latestItem.link))
		}
	}
}

func (session *Session) processUnsubscribeCommand(context *Context, args string) string {
	subscriptions := session.getSubscriptions(context)

	index, err := strconv.Atoi(args)
	if err != nil || index <= 0 || index > len(subscriptions) {
		return fmt.Sprintf(`Invalid index.
            
%s`, session.processListCommand(context))
	}

	index -= 1

	subscription := subscriptions[index]

	if err := session.unsubscribe(context, subscription); err != nil {
		return err.Error()
	} else if err := session.stopObserving(context, subscription); err != nil {
		return err.Error()
	} else {
		return fmt.Sprintf(`%s unsubscribed.`, HTMLLink(subscription.Title, subscription.Link))
	}
}

func (session *Session) processTopSubscriptionsCommand(context *Context, args string) string {
	if statistics, err := session.db.GetTopSubscriptions(5); err != nil {
		return err.Error()
	} else if len(statistics) == 0 {
		return "No enough data."
	} else {
		var message string
		for idx, statistic := range statistics {
			message += fmt.Sprintf("%d. %s (👥 %d)\n", idx+1, HTMLLink(statistic.Subscription.Title, statistic.Subscription.Link), statistic.Count)
		}
		return message
	}
}

// Send & Reply

func (session *Session) send(context *Context, text string, disableWebPagePreview bool) error {
	if context.account.Status == -1 {
		return nil
	}

	message := tgbotapi.MessageConfig{
		BaseChat: tgbotapi.BaseChat{
			ChatID:           context.id,
			ReplyToMessageID: 0,
		},
		Text:                  text,
		ParseMode:             "HTML",
		DisableWebPagePreview: disableWebPagePreview,
	}
	_, err := session.bot.Send(message)
	if err != nil {
		session.processError(context, err)
	}
	return err
}

func (session *Session) reply(context *Context, replyToMessageID int, text string, disableWebPagePreview bool) error {
	if context.account.Status == -1 {
		return nil
	}

	message := tgbotapi.MessageConfig{
		BaseChat: tgbotapi.BaseChat{
			ChatID:           context.id,
			ReplyToMessageID: replyToMessageID,
		},
		Text:                  text,
		ParseMode:             "HTML",
		DisableWebPagePreview: disableWebPagePreview,
	}
	_, err := session.bot.Send(message)
	if err != nil {
		session.processError(context, err)
	}
	return err
}

func (session *Session) processError(context *Context, err error) {
	switch err.Error() {
	case errChatNotFound, errNotMember:
		context.account.Status = -1
		session.db.SaveAccount(context.account)
	}
}

func (session *Session) processFeedItems(context *Context, items []*Item, subscription *Subscription) {
	if len(items) == 0 || subscription == nil {
		return
	}

	var needsUpdate = false

	for id := range context.publishedFeeds[subscription.Id] {
		published := false
		for _, item := range items {
			if item.id == id {
				published = true
				break
			}
		}
		if published {
			continue
		}

		delete(context.publishedFeeds[subscription.Id], id)
		needsUpdate = true
	}

	var newItems []*Item

	for _, item := range items {
		if context.publishedFeeds[subscription.Id][item.id] != nil {
			continue
		}

		context.publishedFeeds[subscription.Id][item.id] = map[string]interface{}{
			"published-timestamp": time.Now().Unix(),
		}

		newItems = append(newItems, item)
		needsUpdate = true
	}

	for _, item := range newItems {
		session.send(context, HTMLLink(item.title, item.link), false)
	}

	if needsUpdate {
		session.db.SetRecentlyPublishedFeeds(context.account, subscription, context.publishedFeeds[subscription.Id])
	}
}

func (session *Session) subscribe(context *Context, channel *Channel) (*Subscription, error) {
	id := channel.id

	subscription := context.subscriptions[id]
	if subscription != nil {
		return nil, fmt.Errorf(`Subscription %s exists`, HTMLLink(subscription.Title, subscription.Link))
	}

	subscription = &Subscription{
		Id:        id,
		Link:      channel.link,
		Title:     channel.title,
		Timestamp: time.Now().Unix(),
	}
	context.subscriptions[id] = subscription

	context.publishedFeeds[id] = make(map[string]interface{})

	err := session.db.AddSubscription(context.account, subscription)
	if err != nil {
		return nil, err
	}

	return subscription, nil
}

func (session *Session) unsubscribe(context *Context, subscription *Subscription) error {
	err := session.db.RemoveSubscription(context.account, subscription)
	if err != nil {
		return err
	}
	delete(context.subscriptions, subscription.Id)

	err = session.db.ClearRecentlyPublishedFeeds(context.account, subscription)
	if err != nil {
		return err
	}
	delete(context.publishedFeeds, subscription.Id)

	return err
}

func (session *Session) updateRecentlyPublishedFeeds(context *Context, subscription *Subscription, items []*Item) error {
	for _, item := range items {
		context.publishedFeeds[subscription.Id][item.id] = map[string]interface{}{
			"published-timestamp": time.Now().Unix(),
		}
	}
	return session.db.SetRecentlyPublishedFeeds(context.account, subscription, context.publishedFeeds[subscription.Id])
}

func (session *Session) getSubscriptions(context *Context) []*Subscription {
	subscriptions := make([]*Subscription, 0)
	for _, subscription := range context.subscriptions {
		subscriptions = append(subscriptions, subscription)
	}

	sort.SliceStable(subscriptions, func(i, j int) bool {
		return subscriptions[i].Timestamp < subscriptions[j].Timestamp
	})

	return subscriptions
}
