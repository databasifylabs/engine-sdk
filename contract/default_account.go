package contract

import (
	"context"

	"github.com/databasifylabs/engine-sdk/dtos"
	"github.com/databasifylabs/engine-sdk/enums"
	"github.com/databasifylabs/engine-sdk/requests"
)

func CreateDefaultAccount(ctx context.Context, accounts Accounts, account *dtos.DefaultAccount) error {
	if account == nil {
		return nil
	}
	if _, err := accounts.CreateDatabase(ctx, requests.DatabaseCreate{Name: account.Database}); err != nil {
		return err
	}
	_, err := accounts.CreateUser(ctx, requests.UserCreate{
		Name: account.User, Password: account.Password,
		Grants: []dtos.Grant{{Database: account.Database, Access: enums.AccessWrite}},
	})

	return err
}
