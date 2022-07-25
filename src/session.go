package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

type Session struct {
	bot     *tgbotapi.BotAPI
	token   string
	handler func(context *Context, update tgbotapi.Update)
}

func InitSession() {
	sessionOnce.Do(func() {
		bot, err := tgbotapi.NewBotAPI(token)
		if err != nil {
			log.Fatal("err")
		}

		u := tgbotapi.NewUpdate(0)
		u.Timeout = 10
		_, err = bot.GetUpdates(u)
		if err != nil {
			log.Fatal(err)
		}

		session = &Session{
			token: token,
			bot:   bot,
		}
	})
	session.Run()
	log.Println(`Session initialized`)
}

func (session *Session) Run() {
	go session.RunLoop()
}

func (session *Session) RunLoop() {
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

		id := message.Chat.ID
		kind := -1
		if message.Chat.IsPrivate() {
			kind = 0
		} else if message.Chat.IsGroup() || message.Chat.IsSuperGroup() {
			kind = 1
		} else if message.Chat.IsChannel() {
			kind = 2
		}

		context := GetCachedContext(id, kind)
		if context == nil {
			account := &Account{
				Id:     id,
				Kind:   kind,
				Status: 1,
			}
			err = fb.SaveAccount(account)
			if err != nil {
				log.Println(err)
				continue
			}
			context = &Context{
				id:            id,
				account:       account,
				subscriptions: make(map[string]*Subscription),
				caches:        make(map[string]map[string]interface{}),
			}
			CacheContext(context)
		}

		session.handleMessage(context, message)
	}
}

func (session *Session) Send(context *Context, message string) error {
	if context.account.Status == -1 {
		return nil
	}

	msg := tgbotapi.NewMessage(context.id, message)
	msg.ParseMode = "markdown"
	_, err := session.bot.Send(msg)
	if err != nil {
		session.handleError(context, err)
	}
	return err
}

func (session *Session) Reply(context *Context, replyToMessageID int, message string) error {
	if context.account.Status == -1 {
		return nil
	}

	msg := tgbotapi.NewMessage(context.id, message)
	msg.ParseMode = "markdown"
	msg.ReplyToMessageID = replyToMessageID
	_, err := session.bot.Send(msg)
	if err != nil {
		session.handleError(context, err)
	}
	return err
}

func (session *Session) handleMessage(context *Context, message *tgbotapi.Message) {
	if context.account.Status == -1 {
		context.account.Status = 1
		fb.SaveAccount(context.account)
	}

	if message.IsCommand() {
		switch strings.ToLower(message.Command()) {
		case "start":
			{
				session.Send(context, "Greetings.")
				break
			}

		case "list":
			{
				response := context.HandleListCommand()
				session.Reply(context, message.MessageID, response)
				break
			}

		case "add", "subscribe", "sub":
			{
				args := message.CommandArguments()
				response := context.HandleSubscribeCommand(args)
				session.Reply(context, message.MessageID, response)
				break
			}

		case "delete", "del", "unsubscribe", "unsub":
			{
				args := message.CommandArguments()
				response := context.HandleUnsubscribeCommand(args)
				session.Reply(context, message.MessageID, response)
				break
			}

		case "hot", "top":
			{
				args := message.CommandArguments()
				response := context.HandleHotCommand(args)
				session.Reply(context, message.MessageID, response)
				break
			}
		default:
			break
		}
	}
}

func (session *Session) handleError(context *Context, err error) {
	if strings.Contains(fmt.Sprint(err), "chat not found") {
		context.account.Status = -1
		fb.SaveAccount(context.account)
	}
}
