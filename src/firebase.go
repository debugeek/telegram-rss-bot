package main

import (
	"context"
	"encoding/base64"
	"log"
	"os"
	"strconv"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Firebase struct {
	app       *firebase.App
	firestore *firestore.Client
	ctx       context.Context
}

func (fb *Firebase) InitDatabase() {
	var conf []byte
	if len(args.FirebaseConf) != 0 {
		conf, _ = base64.StdEncoding.DecodeString(args.FirebaseConf)
	} else if len(args.FirebaseConfEnvKey) != 0 {
		conf, _ = base64.StdEncoding.DecodeString(os.Getenv(args.FirebaseConfEnvKey))
	} else {
		panic("firebase credential not found")
	}
	opt := option.WithCredentialsJSON(conf)

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

	log.Println(`Firebase initialized`)
}

// Account

func (fb *Firebase) GetAccounts() ([]*Account, error) {
	accounts := make([]*Account, 0)

	iter := fb.firestore.Collection("accounts").Documents(fb.ctx)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		var account Account
		doc.DataTo(&account)

		accounts = append(accounts, &account)
	}

	return accounts, nil
}

func (fb *Firebase) GetAccount(id int64) (*Account, error) {
	iter := fb.firestore.Collection("accounts").Where("id", "==", id).Documents(fb.ctx)

	doc, err := iter.Next()
	if err == iterator.Done {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var account *Account
	err = doc.DataTo(&account)

	return account, err
}

func (fb *Firebase) SaveAccount(account *Account) error {
	id := strconv.FormatInt(account.Id, 10)

	_, err := fb.firestore.Collection("accounts").Doc(id).Set(fb.ctx, account)

	return err
}

// Subscription

func (fb *Firebase) GetSubscriptions(account *Account) (map[string]*Subscription, error) {
	id := strconv.FormatInt(account.Id, 10)

	subscriptions := make(map[string]*Subscription)

	iter := fb.firestore.Collection("assets").Doc(id).Collection("subscriptions").Documents(fb.ctx)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		var subscription Subscription
		err = doc.DataTo(&subscription)
		if err != nil {
			return nil, err
		}

		subscriptions[doc.Ref.ID] = &subscription
	}

	return subscriptions, nil
}

func (fb *Firebase) AddSubscription(account *Account, subscription *Subscription) error {
	id := strconv.FormatInt(account.Id, 10)

	subscriptionRef := fb.firestore.Collection("assets").Doc(id).Collection("subscriptions").Doc(subscription.Id)
	statisticRef := fb.firestore.Collection("statistics").Doc("subscriptions").Collection("subscribe_count").Doc(subscription.Id)

	err := fb.firestore.RunTransaction(fb.ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		var statistic SubscriptionStatistic

		doc, err := tx.Get(statisticRef)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				statistic = SubscriptionStatistic{
					Count:        0,
					Subscription: subscription,
				}
			} else {
				return err
			}
		} else {
			doc.DataTo(&statistic)
		}

		statistic.Count++

		err = tx.Set(statisticRef, statistic)
		if err != nil {
			return err
		}

		err = tx.Set(subscriptionRef, subscription)
		if err != nil {
			return err
		}

		return nil
	})

	return err
}

func (fb *Firebase) DeleteSubscription(account *Account, subscription *Subscription) error {
	id := strconv.FormatInt(account.Id, 10)

	subscriptionRef := fb.firestore.Collection("assets").Doc(id).Collection("subscriptions").Doc(subscription.Id)
	statisticRef := fb.firestore.Collection("statistics").Doc("subscriptions").Collection("subscribe_count").Doc(subscription.Id)

	err := fb.firestore.RunTransaction(fb.ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		var statistic SubscriptionStatistic

		doc, err := tx.Get(statisticRef)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				statistic = SubscriptionStatistic{
					Count:        0,
					Subscription: subscription,
				}
			} else {
				return err
			}
		} else {
			doc.DataTo(&statistic)
		}

		statistic.Count--

		if statistic.Count <= 0 {
			err = tx.Delete(statisticRef)
			if err != nil {
				return err
			}
		} else {
			err = tx.Set(statisticRef, statistic)
			if err != nil {
				return err
			}
		}

		err = tx.Delete(subscriptionRef)
		if err != nil {
			return err
		}
		return nil
	})

	return err
}

func (fb *Firebase) GetFeedCache(account *Account, subscription *Subscription) (map[string]interface{}, error) {
	id := strconv.FormatInt(account.Id, 10)

	cache := make(map[string]interface{})

	dsnap, err := fb.firestore.Collection("assets").Doc(id).Collection("caches").Doc(subscription.Id).Get(fb.ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return cache, nil
		} else {
			return nil, err
		}
	} else {
		for k, v := range dsnap.Data() {
			cache[k] = v
		}
		return cache, nil
	}
}

func (fb *Firebase) SetFeedCache(account *Account, subscription *Subscription, cache map[string]interface{}) error {
	id := strconv.FormatInt(account.Id, 10)

	_, err := fb.firestore.Collection("assets").Doc(id).Collection("caches").Doc(subscription.Id).Set(fb.ctx, cache)

	return err
}

func (fb *Firebase) DeleteFeedCache(account *Account, subscription *Subscription) error {
	id := strconv.FormatInt(account.Id, 10)

	_, err := fb.firestore.Collection("assets").Doc(id).Collection("caches").Doc(subscription.Id).Delete(fb.ctx)

	return err
}

// Statistic

func (fb *Firebase) GetTopSubscriptions(num int) ([]*SubscriptionStatistic, error) {
	statistics := make([]*SubscriptionStatistic, 0)

	err := fb.firestore.RunTransaction(fb.ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		querier := fb.firestore.Collection("statistics").Doc("subscriptions").Collection("subscribe_count").OrderBy("count", firestore.Desc).Limit(num)
		iter := tx.Documents(querier)
		for {
			doc, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return err
			}

			var statistic SubscriptionStatistic
			err = doc.DataTo(&statistic)
			if err != nil {
				return err
			}

			statistics = append(statistics, &statistic)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return statistics, nil
}
