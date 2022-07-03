package main

import (
	"context"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go"
)

type Firebase struct {
	app       *firebase.App
	firestore *firestore.Client
	ctx       context.Context
}

func SharedFirebase() Firebase {
	firebaseOnce.Do(func() {
		fb = Firebase{}
		fb.ctx = context.Background()

		if app, err := firebase.NewApp(fb.ctx, nil, opt); err != nil {
			panic(err)
		} else {
			fb.app = app
		}

		if firestore, err := fb.app.Firestore(fb.ctx); err != nil {
			panic(err)
		} else {
			fb.firestore = firestore
		}
	})
	return fb
}
