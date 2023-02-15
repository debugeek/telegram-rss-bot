package main

import (
	"sync"
)

var (
	token string

	sessionOnce sync.Once
	session     *Session

	dbOnce sync.Once
	db     DatabaseProtocol

	monitorOnce sync.Once
	monitor     *Monitor

	contexts map[int64]*Context = make(map[int64]*Context)
)

var errChatNotFound = "Bad Request: chat not found"
var errNotMember = "Forbidden: bot is not a member of the channel chat"
