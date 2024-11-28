package wallet

import (
	"context"
	"database/sql"
	goutils "github.com/mudphilo/go-utils"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/trace"
	"os"
	"strconv"
)

func GetUserTokenAndClient(token string) (tokenString string, clientID int64) {

	prefix := os.Getenv("ACCOUNT_PREFIX")
	UserIDLength := len(prefix)

	if len(token) < UserIDLength {

		return "", 0
	}

	client, _ := strconv.ParseInt(token[0:UserIDLength], 10, 64)
	token = token[UserIDLength:]
	return token, client
}

func GetUserAndClient(accountId string) (userID string, clientID int64) {

	prefix := os.Getenv("ACCOUNT_PREFIX")
	UserIDLength := len(prefix)
	if len(accountId) < UserIDLength {

		return "", 0
	}

	client, _ := strconv.ParseInt(accountId[0:UserIDLength], 10, 64)
	user := accountId[UserIDLength:]

	return user, client
}

func CreateClient(tr trace.Tracer, ctx context.Context, db *sql.DB, client Client) error {

	ctx, span := tr.Start(ctx, "CreateClient")
	defer span.End()

	dbUtils := goutils.Db{DB: db, Context: ctx}

	inserts := map[string]interface{}{
		"account":               client.ID,
		"authentication_header": client.AuthenticationHeader,
		"authentication_string": client.AuthenticationString,
		"base_url":              client.BaseURL,
		"api_version":           client.APIVersion,
	}

	_, err := dbUtils.UpsertWithContext("clients", inserts, []string{"account", "authentication_header", "authentication_string", "base_url", "api_version"})
	if err != nil {

		logrus.WithContext(ctx).
			WithFields(logrus.Fields{
				"description": "error creating  new client",
				"data":        inserts,
			}).
			Error(err.Error())

		return err

	}

	update := map[string]interface{}{
		"api_version": client.APIVersion,
	}

	condition := map[string]interface{}{
		"account": client.ID,
	}

	_, err = dbUtils.UpdateWithContext("clients", condition, update)
	if err != nil {

		logrus.WithContext(ctx).
			WithFields(logrus.Fields{
				"description": "error updating  client",
				"data":        inserts,
			}).
			Error(err.Error())

		return err

	}

	return err

}

func DeleteClient(tr trace.Tracer, ctx context.Context, db *sql.DB, id int64) error {

	ctx, span := tr.Start(ctx, "DeleteClient")
	defer span.End()

	dbUtils := goutils.Db{DB: db, Context: ctx}

	_, err := dbUtils.DeleteWithContext("clients", map[string]interface{}{"account": id})
	if err != nil {

		logrus.WithContext(ctx).
			WithFields(logrus.Fields{
				"description": "error deleting client",
				"data":        id,
			}).
			Error(err.Error())

		return err

	}

	return err

}

func GetClient(tr trace.Tracer, ctx context.Context, db *sql.DB, clientID int64) Client {

	ctx, span := tr.Start(ctx, "GetClient")
	defer span.End()

	query := "SELECT base_url, authentication_header,authentication_string FROM clients WHERE account = ? "
	dbUtils := goutils.Db{DB: db, Context: ctx}
	dbUtils.SetQuery(query)
	dbUtils.SetParams(clientID)

	var base_url, authenticationHeader, authenticationString sql.NullString
	err := dbUtils.FetchOneWithContext().Scan(&base_url, &authenticationHeader, &authenticationString)
	if err != nil {

		logrus.WithContext(ctx).
			WithFields(logrus.Fields{
				"description": "error retrieving client details",
			}).
			Error(err.Error())

		return Client{}

	}

	client := Client{
		ID:                   clientID,
		BaseURL:              base_url.String,
		AuthenticationHeader: authenticationHeader.String,
		AuthenticationString: authenticationString.String,
		APIVersion:           1,
	}

	query = "SELECT api_version FROM clients WHERE account = ? "
	dbUtils.SetQuery(query)
	dbUtils.SetParams(clientID)

	var apiVersion sql.NullInt64
	err = dbUtils.FetchOneWithContext().Scan(&apiVersion)
	if err != nil {

		logrus.WithContext(ctx).
			WithFields(logrus.Fields{
				"description": "error retrieving client api version",
			}).
			Error(err.Error())

		return client

	}

	client.APIVersion = apiVersion.Int64
	return client

}
