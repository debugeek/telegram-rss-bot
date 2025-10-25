package main

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"

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
	return UserData{
		Feeds:      make(map[string]*Feed),
		FeedStatus: map[string]*FeedStatus{},
	}
}

func (app *App) DidLoadUser(session *tgbot.Session[BotData, UserData], user *tgbot.User[UserData]) {
	if user.UserData.Feeds == nil {
		user.UserData.Feeds = make(map[string]*Feed)
	}
	if user.UserData.FeedStatus == nil {
		user.UserData.FeedStatus = make(map[string]*FeedStatus)
		for _, feed := range user.UserData.Feeds {
			user.UserData.FeedStatus[feed.Id] = &FeedStatus{
				PublishedItems: make(map[string]bool),
			}
		}
	}
	for _, feed := range user.UserData.Feeds {
		app.startObserving(session, feed)
	}
}

func (app *App) DidLoadPreference() {

}

func (app *App) processListCommand(session *tgbot.Session[BotData, UserData], args string, message *tgbotapi.Message) tgbot.CmdResult {
	session.SendTextWithConfig(app.formattedFeedList(session), tgbot.MessageConfig{
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

	if feed, items, err := fetchFeed(args); err != nil {
		session.ReplyText(err.Error(), message.MessageID)
	} else if err := app.subscribe(session, feed, items); err != nil {
		session.ReplyText(err.Error(), message.MessageID)
	} else if err := app.startObserving(session, feed); err != nil {
		session.ReplyText(err.Error(), message.MessageID)
	} else {
		if len(items) == 0 {
			session.SendTextWithConfig(
				fmt.Sprintf(
					"%s subscribed.",
					HTMLLink(feed.Title, feed.Link)),
				tgbot.MessageConfig{
					ReplyToMessageID: message.MessageID,
					ParseMode:        tgbot.ParseModeHTML,
				})
		} else {
			latestItem := items[len(items)-1]
			session.SendTextWithConfig(
				fmt.Sprintf(
					"%s subscribed. Here is the latest feed from the feed.\n\n%s",
					HTMLLink(feed.Title, feed.Link),
					HTMLLink(latestItem.Title, latestItem.Link)),
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
				app.formattedFeedList(session)),
			tgbot.MessageConfig{
				ReplyToMessageID: message.MessageID,
				ParseMode:        tgbot.ParseModeHTML,
			})
		return tgbot.CmdResultWaitingForInput
	}

	index, err := strconv.Atoi(args)
	feeds := app.getFeeds(session)
	if err != nil || index <= 0 || index > len(feeds) {
		session.SendTextWithConfig(
			fmt.Sprintf(
				"Send me a valid index to unsubscribe.\n\n%s",
				app.formattedFeedList(session)),
			tgbot.MessageConfig{
				ReplyToMessageID: message.MessageID,
				ParseMode:        tgbot.ParseModeHTML,
			})
		return tgbot.CmdResultWaitingForInput
	}

	index -= 1

	feed := feeds[index]
	if err := app.unsubscribe(session, feed); err != nil {
		session.ReplyText(err.Error(), message.MessageID)
	} else if err := app.stopObserving(session, feed); err != nil {
		session.ReplyText(err.Error(), message.MessageID)
	} else {
		session.SendTextWithConfig(
			fmt.Sprintf("%s unsubscribed.", HTMLLink(feed.Title, feed.Link)),
			tgbot.MessageConfig{
				ReplyToMessageID: message.MessageID,
				ParseMode:        tgbot.ParseModeHTML,
			})
	}
	return tgbot.CmdResultProcessed
}

func (app *App) processTopCommand(session *tgbot.Session[BotData, UserData], args string, message *tgbotapi.Message) tgbot.CmdResult {
	counter := make(map[string]int)
	lookup := make(map[string]*Feed)

	for _, s := range app.bot.Client.Sessions {
		for _, feed := range s.User.UserData.Feeds {
			counter[feed.Id]++
			lookup[feed.Id] = feed
		}
	}

	type Candidate struct {
		feed  *Feed
		count int
	}
	var candidates []Candidate
	for id, count := range counter {
		candidates = append(candidates, Candidate{
			feed:  lookup[id],
			count: count,
		})
	}
	if len(candidates) == 0 {
		session.ReplyText("No enough data.", message.MessageID)
		return tgbot.CmdResultProcessed
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].count > candidates[j].count
	})

	var text string
	for idx, candidate := range candidates {
		text += fmt.Sprintf("%d. %s (👥 %d)\n",
			idx+1,
			HTMLLink(candidate.feed.Title, candidate.feed.Link),
			candidate.count)
	}
	session.SendTextWithConfig(text, tgbot.MessageConfig{
		ReplyToMessageID: message.MessageID,
		ParseMode:        tgbot.ParseModeHTML,
	})

	return tgbot.CmdResultProcessed
}

func (app *App) processSetTopicCommand(session *tgbot.Session[BotData, UserData], args string, message *tgbotapi.Message) tgbot.CmdResult {
	if _, ok := session.CommandSession.Args["index"]; !ok {
		index, err := strconv.Atoi(args)
		if err != nil || index <= 0 || index > len(app.getFeeds(session)) {
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

	feeds := app.getFeeds(session)
	if index <= 0 || index > len(feeds) {
		return tgbot.CmdResultProcessed
	}
	feed := feeds[index-1]
	session.User.UserData.Feeds[feed.Id].Topic = topic
	app.firebase.UpdateUser(session.User)

	session.SendTextWithConfig(fmt.Sprintf("Items from %s will now be sent to topic %d.", feed.Title, topic), tgbot.MessageConfig{
		ReplyToMessageID: message.MessageID,
	})

	return tgbot.CmdResultProcessed
}

func (app *App) getFeeds(session *tgbot.Session[BotData, UserData]) []*Feed {
	feeds := make([]*Feed, 0)
	for _, feed := range session.User.UserData.Feeds {
		feeds = append(feeds, feed)
	}

	sort.SliceStable(feeds, func(i, j int) bool {
		return feeds[i].SubscribedTime.Before(feeds[j].SubscribedTime)
	})

	return feeds
}

func (app *App) formattedFeedList(session *tgbot.Session[BotData, UserData]) string {
	feeds := app.getFeeds(session)
	if len(feeds) == 0 {
		return "Your list is empty."
	}
	var message string
	for idx, feed := range feeds {
		message += fmt.Sprintf("%d. %s \n", idx+1, HTMLLink(feed.Title, feed.Link))
	}
	return message
}

func (app *App) subscribe(session *tgbot.Session[BotData, UserData], feed *Feed, items []*Item) error {
	if session.User.UserData.Feeds[feed.Id] != nil {
		return fmt.Errorf("Feed %s exists", HTMLLink(feed.Title, feed.Link))
	}

	latestPublishedTime := items[0].PublishedTime
	for _, item := range items[1:] {
		if item.PublishedTime.After(latestPublishedTime) {
			latestPublishedTime = item.PublishedTime
		}
	}

	session.User.UserData.Feeds[feed.Id] = feed
	session.User.UserData.FeedStatus[feed.Id] = &FeedStatus{
		PublishedItems:      make(map[string]bool),
		LatestPublishedTime: latestPublishedTime,
	}
	return app.firebase.UpdateUser(session.User)
}

func (app *App) unsubscribe(session *tgbot.Session[BotData, UserData], feed *Feed) error {
	delete(session.User.UserData.Feeds, feed.Id)
	delete(session.User.UserData.FeedStatus, feed.Id)
	return app.firebase.UpdateUser(session.User)
}

func (app *App) startObserving(session *tgbot.Session[BotData, UserData], feed *Feed) error {
	observer := &Observer{
		identifier: session.ID,
		handler: func(items []*Item) {
			app.processFeedItems(session, items, feed)
		},
	}
	app.monitor.addObserver(observer, feed.Link)
	return nil
}

func (app *App) stopObserving(session *tgbot.Session[BotData, UserData], feed *Feed) error {
	app.monitor.removeObserver(session.User.ID, feed.Link)
	return nil
}

func (app *App) processFeedItems(session *tgbot.Session[BotData, UserData], items []*Item, feed *Feed) {
	if len(items) == 0 || feed == nil {
		return
	}

	var needsUpdate = false

	itemIDs := make(map[string]bool, len(items))
	for _, item := range items {
		itemIDs[item.Id] = true
	}
	for id := range session.User.UserData.FeedStatus[feed.Id].PublishedItems {
		if _, exists := itemIDs[id]; !exists {
			delete(session.User.UserData.FeedStatus[feed.Id].PublishedItems, id)
			needsUpdate = true
		}
	}

	var newItems []*Item

	for _, item := range items {
		if session.User.UserData.FeedStatus[feed.Id].PublishedItems[item.Id] {
			continue
		}

		if item.PublishedTime.Before(session.User.UserData.FeedStatus[feed.Id].LatestPublishedTime) {
			continue
		}

		session.User.UserData.FeedStatus[feed.Id].LatestPublishedTime = item.PublishedTime
		session.User.UserData.FeedStatus[feed.Id].PublishedItems[item.Id] = true

		newItems = append(newItems, item)
		needsUpdate = true
	}

	for _, item := range newItems {
		session.SendTextWithConfig(HTMLLink(item.Title, item.Link), tgbot.MessageConfig{
			ParseMode:        tgbot.ParseModeHTML,
			ReplyToMessageID: feed.Topic,
		})
	}

	if needsUpdate {
		app.firebase.UpdateUser(session.User)
	}
}
