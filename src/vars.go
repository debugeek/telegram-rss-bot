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
