package main

import (
	"sync"

	"google.golang.org/api/option"
)

var (
	token string

	opt option.ClientOption

	sessionOnce sync.Once
	session     *Session

	firebaseOnce sync.Once
	fb           Firebase

	monitorOnce sync.Once
	monitor     *Monitor

	contexts map[int64]*Context = make(map[int64]*Context)
)

var errChatNotFound = "Bad Request: chat not found"
var errNotMember = "Forbidden: bot is not a member of the channel chat"
