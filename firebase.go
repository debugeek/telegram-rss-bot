package main

import (
	tgbot "github.com/debugeek/telegram-bot"
)

type Firebase struct {
	tgbot.Firebase[BotData, UserData]
}
