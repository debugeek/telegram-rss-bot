package main

import (
	"strconv"

	"google.golang.org/api/iterator"
)

func (fb Firebase) GetAccounts() ([]*Account, error) {
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

func (fb Firebase) GetAccount(id int64) (*Account, error) {
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

func (fb Firebase) SaveAccount(account *Account) error {
	id := strconv.FormatInt(account.Id, 10)

	_, err := fb.firestore.Collection("accounts").Doc(id).Set(fb.ctx, account)

	return err
}
