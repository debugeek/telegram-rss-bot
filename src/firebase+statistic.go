package main

import (
	"context"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
)

func (fb Firebase) GetTopSubscriptions(num int) ([]*SubscriptionStatistic, error) {
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
