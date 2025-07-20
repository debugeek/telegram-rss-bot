package main

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/alexflint/go-arg"
	tgbot "github.com/debugeek/telegram-bot"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

var args struct {
	TelegramBotToken      string `arg:"-t,--tgbot-token" help:"telegram bot token"`
	TelegramBotTokenKey   string `arg:"--tgbot-token-key" help:"env key for telegram bot token"`
	FirebaseCredential    string `arg:"-c,--firebase-credential" help:"base64 encoded firebase credential"`
	FirebaseCredentialKey string `arg:"--firebase-credential-key" help:"env key for base64 encoded firebase credential"`
	FirebaseDatabaseURL   string `arg:"-d,--firebase-database" help:"firebase database url"`
}

type App struct {
	bot      *tgbot.TgBot[BotData, UserData]
	firebase Firebase
	monitor  *Monitor
}

func (app *App) launch() {
	arg.MustParse(&args)

	telegramBotToken := args.TelegramBotToken
	if telegramBotToken == "" {
		telegramBotToken = os.Getenv(args.TelegramBotTokenKey)
	}
	if telegramBotToken == "" {
		panic(errors.New(errTelegramBotTokenNotFound))
	}

	encodedFirebaseCredential := args.FirebaseCredential
	if encodedFirebaseCredential == "" {
		encodedFirebaseCredential = os.Getenv(args.FirebaseCredentialKey)
	}
	if encodedFirebaseCredential == "" {
		panic(errors.New(errFirebaseCredentialNotFound))
	}
	firebaseCredential, err := base64.StdEncoding.DecodeString(encodedFirebaseCredential)
	if err != nil {
		panic(err)
	}

	firebaseDatabaseURL := args.FirebaseDatabaseURL
	if firebaseDatabaseURL == "" {
		panic(errors.New(errFirebaseDatabaseNotFound))
	}

	bot := tgbot.NewBot(tgbot.Config{
		TelegramBotToken:    telegramBotToken,
		FirebaseCredential:  firebaseCredential,
		FirebaseDatabaseURL: firebaseDatabaseURL,
	}, app)
	bot.RegisterCommandHandler(CmdList, app.processListCommand)
	bot.RegisterCommandHandler(CmdAdd, app.processAddCommand)
	bot.RegisterCommandHandler(CmdDelete, app.processDeleteCommand)
	bot.RegisterCommandHandler(CmdTop, app.processTopCommand)
	bot.RegisterCommandHandler(CmdSetTopic, app.processSetTopicCommand)

	app.bot = bot

	app.firebase = Firebase{
		Firebase: bot.Client.Firebase,
	}

	app.monitor = &Monitor{
		observers: make(map[string]map[int64]*Observer),
	}

	bot.Start()

	go app.monitor.runLoop()
}

func (app *App) NewUserData() UserData {
	return UserData{}
}

func (app *App) DidLoadUser(session *tgbot.Session[BotData, UserData], user *tgbot.User[UserData]) {
	user.UserData.subscriptions = make(map[string]*Subscription)
	user.UserData.publishedFeeds = make(map[string]map[string]interface{})

	if subscriptions, err := app.firebase.GetSubscriptions(user.ID); err == nil {
		for id, subscription := range subscriptions {
			user.UserData.subscriptions[id] = subscription

			if publishedFeeds, err := app.firebase.GetRecentlyPublishedFeeds(user.ID, subscription); err == nil {
				user.UserData.publishedFeeds[id] = publishedFeeds
			}

			app.startObserving(session, subscription)
		}
	}
}

func (app *App) DidLoadPreference() {

}

func (app *App) processListCommand(session *tgbot.Session[BotData, UserData], args string, message *tgbotapi.Message) tgbot.CmdResult {
	session.SendTextWithConfig(app.formattedSubscriptionList(session), tgbot.MessageConfig{
		ReplyToMessageID: message.MessageID,
		ParseMode:        tgbot.ParseModeHTML,
	})
	return tgbot.CmdResultProcessed
}

func (app *App) processAddCommand(session *tgbot.Session[BotData, UserData], args string, message *tgbotapi.Message) tgbot.CmdResult {
	if len(args) == 0 {
		session.ReplyText("Send me a link to subscribe.", message.MessageID)
		return tgbot.CmdResultWaitingForInput
	}

	if channel, items, err := fetchItems(args); err != nil {
		session.ReplyText(err.Error(), message.MessageID)
	} else if subscription, err := app.subscribe(session, channel); err != nil {
		session.ReplyText(err.Error(), message.MessageID)
	} else if err := app.updateRecentlyPublishedFeeds(session, subscription, items); err != nil {
		session.ReplyText(err.Error(), message.MessageID)
	} else if err := app.startObserving(session, subscription); err != nil {
		session.ReplyText(err.Error(), message.MessageID)
	} else {
		if len(items) == 0 {
			session.SendTextWithConfig(
				fmt.Sprintf(
					"%s subscribed.",
					HTMLLink(subscription.Title, subscription.Link)),
				tgbot.MessageConfig{
					ReplyToMessageID: message.MessageID,
					ParseMode:        tgbot.ParseModeHTML,
				})
		} else {
			latestItem := items[len(items)-1]
			session.SendTextWithConfig(
				fmt.Sprintf(
					"%s subscribed. Here is the latest feed from the channel.\n\n%s",
					HTMLLink(subscription.Title, subscription.Link),
					HTMLLink(latestItem.title, latestItem.link)),
				tgbot.MessageConfig{
					ReplyToMessageID: message.MessageID,
					ParseMode:        tgbot.ParseModeHTML,
				})
		}
	}
	return tgbot.CmdResultProcessed
}

func (app *App) processDeleteCommand(session *tgbot.Session[BotData, UserData], args string, message *tgbotapi.Message) tgbot.CmdResult {
	if len(args) == 0 {
		session.SendTextWithConfig(
			fmt.Sprintf(
				"Send me an index to unsubscribe.\n\n%s",
				app.formattedSubscriptionList(session)),
			tgbot.MessageConfig{
				ReplyToMessageID: message.MessageID,
				ParseMode:        tgbot.ParseModeHTML,
			})
		return tgbot.CmdResultWaitingForInput
	}

	index, err := strconv.Atoi(args)
	subscriptions := app.getSubscriptions(session)
	if err != nil || index <= 0 || index > len(subscriptions) {
		session.SendTextWithConfig(
			fmt.Sprintf(
				"Send me a valid index to unsubscribe.\n\n%s",
				app.formattedSubscriptionList(session)),
			tgbot.MessageConfig{
				ReplyToMessageID: message.MessageID,
				ParseMode:        tgbot.ParseModeHTML,
			})
		return tgbot.CmdResultWaitingForInput
	}

	index -= 1

	subscription := subscriptions[index]
	if err := app.unsubscribe(session, subscription); err != nil {
		session.ReplyText(err.Error(), message.MessageID)
	} else if err := app.stopObserving(session, subscription); err != nil {
		session.ReplyText(err.Error(), message.MessageID)
	} else {
		session.SendTextWithConfig(
			fmt.Sprintf("%s unsubscribed.", HTMLLink(subscription.Title, subscription.Link)),
			tgbot.MessageConfig{
				ReplyToMessageID: message.MessageID,
				ParseMode:        tgbot.ParseModeHTML,
			})
	}
	return tgbot.CmdResultProcessed
}

func (app *App) processTopCommand(session *tgbot.Session[BotData, UserData], args string, message *tgbotapi.Message) tgbot.CmdResult {
	if statistics, err := app.firebase.GetTopSubscriptions(5); err != nil {
		session.ReplyText(err.Error(), message.MessageID)
	} else if len(statistics) == 0 {
		session.ReplyText("No enough data.", message.MessageID)
	} else {
		var text string
		for idx, statistic := range statistics {
			text += fmt.Sprintf("%d. %s (👥 %d)\n",
				idx+1,
				HTMLLink(statistic.Subscription.Title, statistic.Subscription.Link),
				statistic.Count)
		}
		session.SendTextWithConfig(text, tgbot.MessageConfig{
			ReplyToMessageID: message.MessageID,
			ParseMode:        tgbot.ParseModeHTML,
		})
	}
	return tgbot.CmdResultProcessed
}

func (app *App) processSetTopicCommand(session *tgbot.Session[BotData, UserData], args string, message *tgbotapi.Message) tgbot.CmdResult {
	if _, ok := session.CommandSession.Args["index"]; !ok {
		index, err := strconv.Atoi(args)
		if err != nil || index <= 0 || index > len(app.getSubscriptions(session)) {
			session.SendTextWithConfig("Send me a valid index number.", tgbot.MessageConfig{
				ReplyToMessageID: message.MessageID,
				ParseMode:        tgbot.ParseModeHTML,
			})
			return tgbot.CmdResultWaitingForInput
		}
		session.CommandSession.Args["index"] = index
		session.SendTextWithConfig("Now send me the new topic.", tgbot.MessageConfig{
			ReplyToMessageID: message.MessageID,
			ParseMode:        tgbot.ParseModeHTML,
		})
		return tgbot.CmdResultWaitingForInput
	}

	if _, ok := session.CommandSession.Args["topic"]; !ok {
		topic, err := strconv.Atoi(args)
		if err != nil || topic < 0 {
			session.SendTextWithConfig("Topic cannot be empty. Please send a valid topic.", tgbot.MessageConfig{
				ReplyToMessageID: message.MessageID,
			})
			return tgbot.CmdResultWaitingForInput
		}

		session.CommandSession.Args["topic"] = topic
	}

	indexVal, indexOk := session.CommandSession.Args["index"]
	topicVal, topicOk := session.CommandSession.Args["topic"]

	if !indexOk || !topicOk {
		return tgbot.CmdResultProcessed
	}

	index, ok := indexVal.(int)
	if !ok {
		return tgbot.CmdResultProcessed
	}

	topic, ok := topicVal.(int)
	if !ok {
		return tgbot.CmdResultProcessed
	}

	subscriptions := app.getSubscriptions(session)
	if index <= 0 || index > len(subscriptions) {
		return tgbot.CmdResultProcessed
	}
	subscription := subscriptions[index-1]
	subscription.Topic = topic
	app.firebase.UpdateSubscription(session.User.ID, subscription)

	session.SendTextWithConfig(fmt.Sprintf("Feeds from %s will now be sent to topic %d.", subscription.Title, topic), tgbot.MessageConfig{
		ReplyToMessageID: message.MessageID,
	})

	return tgbot.CmdResultProcessed
}

func (app *App) getSubscriptions(session *tgbot.Session[BotData, UserData]) []*Subscription {
	subscriptions := make([]*Subscription, 0)
	for _, subscription := range session.User.UserData.subscriptions {
		subscriptions = append(subscriptions, subscription)
	}

	sort.SliceStable(subscriptions, func(i, j int) bool {
		return subscriptions[i].Timestamp < subscriptions[j].Timestamp
	})

	return subscriptions
}

func (app *App) formattedSubscriptionList(session *tgbot.Session[BotData, UserData]) string {
	subscriptions := app.getSubscriptions(session)
	if len(subscriptions) == 0 {
		return "Your list is empty."
	}
	var message string
	for idx, subscription := range subscriptions {
		message += fmt.Sprintf("%d. %s \n", idx+1, HTMLLink(subscription.Title, subscription.Link))
	}
	return message
}

func (app *App) updateRecentlyPublishedFeeds(session *tgbot.Session[BotData, UserData], subscription *Subscription, items []*Item) error {
	for _, item := range items {
		session.User.UserData.publishedFeeds[subscription.Id][item.id] = map[string]interface{}{
			"published-timestamp": time.Now().Unix(),
		}
	}
	return app.firebase.SetRecentlyPublishedFeeds(session.User.ID, subscription, session.User.UserData.publishedFeeds[subscription.Id])
}

func (app *App) subscribe(session *tgbot.Session[BotData, UserData], channel *Channel) (*Subscription, error) {
	id := channel.id

	subscription := session.User.UserData.subscriptions[id]
	if subscription != nil {
		return nil, fmt.Errorf("Subscription %s exists", HTMLLink(subscription.Title, subscription.Link))
	}

	subscription = &Subscription{
		Id:        id,
		Link:      channel.link,
		Title:     channel.title,
		Timestamp: time.Now().Unix(),
	}
	session.User.UserData.subscriptions[id] = subscription
	session.User.UserData.publishedFeeds[id] = make(map[string]interface{})

	err := app.firebase.AddSubscription(session.User.ID, subscription)
	if err != nil {
		return nil, err
	}

	return subscription, nil
}

func (app *App) unsubscribe(session *tgbot.Session[BotData, UserData], subscription *Subscription) error {
	err := app.firebase.RemoveSubscription(session.User.ID, subscription)
	if err != nil {
		return err
	}
	delete(session.User.UserData.subscriptions, subscription.Id)

	err = app.firebase.ClearRecentlyPublishedFeeds(session.User.ID, subscription)
	if err != nil {
		return err
	}
	delete(session.User.UserData.publishedFeeds, subscription.Id)

	return err
}

func (app *App) startObserving(session *tgbot.Session[BotData, UserData], subscription *Subscription) error {
	observer := &Observer{
		identifier: session.ID,
		handler: func(items []*Item) {
			app.processFeedItems(session, items, subscription)
		},
	}
	app.monitor.addObserver(observer, subscription.Link)
	return nil
}

func (app *App) stopObserving(session *tgbot.Session[BotData, UserData], subscription *Subscription) error {
	app.monitor.removeObserver(session.User.ID, subscription.Link)
	return nil
}

func (app *App) processFeedItems(session *tgbot.Session[BotData, UserData], items []*Item, subscription *Subscription) {
	if len(items) == 0 || subscription == nil {
		return
	}

	var needsUpdate = false

	for id := range session.User.UserData.publishedFeeds[subscription.Id] {
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

		delete(session.User.UserData.publishedFeeds[subscription.Id], id)
		needsUpdate = true
	}

	var newItems []*Item

	for _, item := range items {
		if session.User.UserData.publishedFeeds[subscription.Id][item.id] != nil {
			continue
		}

		session.User.UserData.publishedFeeds[subscription.Id][item.id] = map[string]interface{}{
			"published-timestamp": time.Now().Unix(),
		}

		newItems = append(newItems, item)
		needsUpdate = true
	}

	for _, item := range newItems {
		session.SendTextWithConfig(HTMLLink(item.title, item.link), tgbot.MessageConfig{
			ParseMode:        tgbot.ParseModeHTML,
			ReplyToMessageID: subscription.Topic,
		})
	}

	if needsUpdate {
		app.firebase.SetRecentlyPublishedFeeds(session.User.ID, subscription, session.User.UserData.publishedFeeds[subscription.Id])
	}
}
