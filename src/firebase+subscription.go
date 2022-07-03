package main

import (
	"context"
	"strconv"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (fb Firebase) GetSubscriptions(account *Account) (map[string]*Subscription, error) {
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

func (fb Firebase) AddSubscription(account *Account, subscription *Subscription) error {
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

func (fb Firebase) DeleteSubscription(account *Account, subscription *Subscription) error {
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

func (fb Firebase) GetFeedCache(account *Account, subscription *Subscription) (map[string]interface{}, error) {
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

func (fb Firebase) SetFeedCache(account *Account, subscription *Subscription, cache map[string]interface{}) error {
	id := strconv.FormatInt(account.Id, 10)

	_, err := fb.firestore.Collection("assets").Doc(id).Collection("caches").Doc(subscription.Id).Set(fb.ctx, cache)

	return err
}

func (fb Firebase) DeleteFeedCache(account *Account, subscription *Subscription) error {
	id := strconv.FormatInt(account.Id, 10)

	_, err := fb.firestore.Collection("assets").Doc(id).Collection("caches").Doc(subscription.Id).Delete(fb.ctx)

	return err
}
