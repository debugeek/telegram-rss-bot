package main

import (
	"log"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

type Session struct {
	bot     *tgbotapi.BotAPI
	token   string
	handler func(s *Session, update tgbotapi.Update)
}

func SharedSession() *Session {
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
	return session
}

func InitSession() {
	SharedSession().Run()
	log.Println(`Session initialized`)
}

func (session *Session) SetHandler(handler func(s *Session, update tgbotapi.Update)) {
	session.handler = handler
}

func (session *Session) Run() {
	session.SetHandler(func(s *Session, update tgbotapi.Update) {
		log.Println(update.Message.Text)

		id := update.Message.Chat.ID

		kind := 0
		if update.Message.Chat.IsPrivate() {
			kind = 0
		} else if update.Message.Chat.IsGroup() || update.Message.Chat.IsSuperGroup() {
			kind = 1
		} else if update.Message.Chat.IsChannel() {
			kind = 2
		}

		context, err := NewContext(id, kind)
		if err != nil {
			log.Println(err)
			return
		}

		if update.Message.IsCommand() {
			switch update.Message.Command() {
			case "start":
				{
					session.Send(context.id, "Greetings.")
					break
				}

			case "list":
				{
					response := context.HandleListCommand()
					session.Reply(context.id, update.Message.MessageID, response)
					break
				}

			case "add", "subscribe":
				{
					args := update.Message.CommandArguments()
					response := context.HandleSubscribeCommand(args)
					session.Reply(context.id, update.Message.MessageID, response)
					break
				}

			case "delete", "unsubscribe":
				{
					args := update.Message.CommandArguments()
					response := context.HandleUnsubscribeCommand(args)
					session.Reply(context.id, update.Message.MessageID, response)
					break
				}

			case "hot", "top":
				{
					args := update.Message.CommandArguments()
					response := context.HandleHotCommand(args)
					session.Reply(context.id, update.Message.MessageID, response)
					break
				}
			default:
				break
			}
		}
	})

	go session.Schedule()
}

func (session *Session) Schedule() {
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
		if update.Message == nil {
			continue
		}

		session.handler(session, update)
	}
}

func (session *Session) Send(chatID int64, message string) error {
	msg := tgbotapi.NewMessage(chatID, message)
	msg.ParseMode = "markdown"
	_, err := session.bot.Send(msg)
	return err
}

func (session *Session) Reply(chatID int64, replyToMessageID int, message string) error {
	msg := tgbotapi.NewMessage(chatID, message)
	msg.ParseMode = "markdown"
	msg.ReplyToMessageID = replyToMessageID
	_, err := session.bot.Send(msg)
	return err
}
