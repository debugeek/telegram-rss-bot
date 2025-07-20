package main

import (
	"context"
	"strconv"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	tgbot "github.com/debugeek/telegram-bot"
)

type Firebase struct {
	tgbot.Firebase[BotData, UserData]
}

// Subscription

func (fb *Firebase) GetSubscriptions(userID int64) (map[string]*Subscription, error) {
	id := strconv.FormatInt(userID, 10)

	subscriptions := make(map[string]*Subscription)

	iter := fb.Firestore.Collection("assets").Doc(id).Collection("subscriptions").Documents(fb.Context)
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

func (fb *Firebase) AddSubscription(userID int64, subscription *Subscription) error {
	id := strconv.FormatInt(userID, 10)

	subscriptionRef := fb.Firestore.Collection("assets").Doc(id).Collection("subscriptions").Doc(subscription.Id)
	statisticRef := fb.Firestore.Collection("statistics").Doc("subscriptions").Collection("subscribers_count").Doc(subscription.Id)

	err := fb.Firestore.RunTransaction(fb.Context, func(ctx context.Context, tx *firestore.Transaction) error {
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

func (fb *Firebase) UpdateSubscription(userID int64, subscription *Subscription) error {
	id := strconv.FormatInt(userID, 10)

	_, err := fb.Firestore.Collection("assets").Doc(id).Collection("subscriptions").Doc(subscription.Id).Set(fb.Context, subscription)

	return err
}

func (fb *Firebase) RemoveSubscription(userID int64, subscription *Subscription) error {
	id := strconv.FormatInt(userID, 10)

	subscriptionRef := fb.Firestore.Collection("assets").Doc(id).Collection("subscriptions").Doc(subscription.Id)
	statisticRef := fb.Firestore.Collection("statistics").Doc("subscriptions").Collection("subscribers_count").Doc(subscription.Id)

	err := fb.Firestore.RunTransaction(fb.Context, func(ctx context.Context, tx *firestore.Transaction) error {
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

func (fb *Firebase) GetRecentlyPublishedFeeds(userID int64, subscription *Subscription) (map[string]interface{}, error) {
	id := strconv.FormatInt(userID, 10)

	cache := make(map[string]interface{})

	dsnap, err := fb.Firestore.Collection("assets").Doc(id).Collection("published_feeds").Doc(subscription.Id).Get(fb.Context)
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

func (fb *Firebase) SetRecentlyPublishedFeeds(userID int64, subscription *Subscription, cache map[string]interface{}) error {
	id := strconv.FormatInt(userID, 10)

	_, err := fb.Firestore.Collection("assets").Doc(id).Collection("published_feeds").Doc(subscription.Id).Set(fb.Context, cache)

	return err
}

func (fb *Firebase) ClearRecentlyPublishedFeeds(userID int64, subscription *Subscription) error {
	id := strconv.FormatInt(userID, 10)

	_, err := fb.Firestore.Collection("assets").Doc(id).Collection("published_feeds").Doc(subscription.Id).Delete(fb.Context)

	return err
}

// Statistic

func (fb *Firebase) GetTopSubscriptions(num int) ([]*SubscriptionStatistic, error) {
	statistics := make([]*SubscriptionStatistic, 0)

	err := fb.Firestore.RunTransaction(fb.Context, func(ctx context.Context, tx *firestore.Transaction) error {
		querier := fb.Firestore.Collection("statistics").Doc("subscriptions").Collection("subscribers_count").OrderBy("count", firestore.Desc).Limit(num)
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
