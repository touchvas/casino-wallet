package wallet

import (
	"bytes"
	"context"
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis"
	"github.com/google/uuid"
	goutils "github.com/mudphilo/go-utils"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/trace"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

var (
	once      sync.Once
	netClient *http.Client
)

func GenerateToken(redisConn *redis.Client, profileID string) string {

	token := uuid.New().String()
	SetRedisKeyWithExpiry(redisConn, token, profileID, 60*60*5)

	sessionKeys := fmt.Sprintf("session:%s", profileID)
	SetRedisKeyWithExpiry(redisConn, sessionKeys, token, 60*60*5)

	return token
}

func GetSessionID(redisConn *redis.Client, profileID string) string {

	sessionKeys := fmt.Sprintf("session:%s", profileID)
	profile, _ := GetRedisKey(redisConn, sessionKeys)
	return profile
}

func GetProfileIDFromtoken(redisConn *redis.Client, token string) string {

	profile, _ := GetRedisKey(redisConn, token)
	return profile
}

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
	}

	_, err := dbUtils.UpsertWithContext("clients", inserts, []string{"account", "authentication_header", "authentication_string", "base_url"})
	if err != nil {

		logrus.WithContext(ctx).
			WithFields(logrus.Fields{
				"description": "error creating client",
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

	return Client{
		ID:                   clientID,
		BaseURL:              base_url.String,
		AuthenticationHeader: authenticationHeader.String,
		AuthenticationString: authenticationString.String,
	}

}

func GetWalletProfile(tr trace.Tracer, ctx context.Context, client Client, profileID string) (*WalletProfile, error) {

	ctx, span := tr.Start(ctx, "GetWalletProfile")
	defer span.End()

	spanID := span.SpanContext().SpanID().String()
	traceID := span.SpanContext().TraceID().String()

	headers := map[string]string{
		client.AuthenticationHeader: client.AuthenticationString,
		"span-id":                   spanID,
		"trace-id":                  traceID,
	}

	profileRequest := ProfileRequest{
		PlayerID: profileID,
		SpanID:   spanID,
		TraceID:  traceID,
	}

	endpoint := fmt.Sprintf("%s/profile", client.BaseURL)

	status, response := HTTPPost(ctx, endpoint, headers, profileRequest)

	if status > 299 || status < 200 {

		logrus.WithContext(ctx).
			WithFields(logrus.Fields{
				"description":      fmt.Sprintf("invalid status %d geting user ", status),
				"endpoint":         endpoint,
				"request":          profileRequest,
				"response_status":  status,
				"response_payload": response,
			}).
			Warn(response)

		return nil, fmt.Errorf("%s", response)

	} else {

		logrus.WithContext(ctx).
			WithFields(logrus.Fields{
				"endpoint":         endpoint,
				"request":          profileRequest,
				"response_status":  status,
				"response_payload": response,
			}).
			Info(response)
	}

	prof := new(WalletProfile)

	err := json.Unmarshal([]byte(response), prof)
	if err != nil {

		logrus.WithContext(ctx).
			WithFields(logrus.Fields{
				"description": "error unmarshalling profile to JSON",
				"data":        response,
			}).
			Error(err.Error())

		return nil, fmt.Errorf("internal server error")

	}

	id := fmt.Sprintf("%d%s", client.ID, prof.ID)

	prof.ID = id

	return prof, nil

}

func DebitWalletProfile(tr trace.Tracer, ctx context.Context, client Client, debit Debit) (*DebitTransactionResponse, error) {

	ctx, span := tr.Start(ctx, "DebitWalletProfile")
	defer span.End()

	spanID := span.SpanContext().SpanID().String()
	traceID := span.SpanContext().TraceID().String()

	providerID, _ := strconv.ParseInt(os.Getenv("PROVIDER_ID"), 10, 64)
	providerName := os.Getenv("PROVIDER_NAME")

	headers := map[string]string{
		client.AuthenticationHeader: client.AuthenticationString,
		"span-id":                   spanID,
		"trace-id":                  traceID,
	}

	debitRequest := DebitRequest{
		PlayerID:      debit.PlayerID,
		ProviderID:    providerID,
		ProviderName:  providerName,
		GameName:      debit.GameName,
		GameID:        debit.GameID,
		TransactionID: debit.TransactionID,
		Amount:        debit.Amount,
		SessionID:     debit.SessionID,
		RoundID:       debit.RoundID,
		SpanID:        spanID,
		TraceID:       traceID,
	}

	endpoint := fmt.Sprintf("%s/debit", client.BaseURL)

	status, response := HTTPPost(ctx, endpoint, headers, debitRequest)
	if status > 299 || status < 200 {

		logrus.WithContext(ctx).
			WithFields(logrus.Fields{
				"description":      fmt.Sprintf("invalid status %d debiting user ", status),
				"endpoint":         endpoint,
				"request":          debitRequest,
				"response_status":  status,
				"response_payload": response,
			}).
			Warn(response)

		if status == http.StatusPaymentRequired {

			prof := new(DebitTransactionResponse)
			prof.Status = http.StatusPaymentRequired
			prof.Description = response
			return prof, nil

		}

		if status == http.StatusConflict {

			prof := new(DebitTransactionResponse)
			prof.Status = http.StatusConflict
			prof.Description = response
			return prof, nil
		}

		return nil, fmt.Errorf("%s", response)

	} else {

		logrus.WithContext(ctx).
			WithFields(logrus.Fields{
				"endpoint":         endpoint,
				"request":          debitRequest,
				"response_status":  status,
				"response_payload": response,
			}).
			Info(response)
	}

	prof := new(DebitTransactionResponse)

	err := json.Unmarshal([]byte(response), prof)
	if err != nil {

		logrus.WithContext(ctx).
			WithFields(logrus.Fields{
				"description": "error unmarshalling TransactionResponse from JSON",
				"data":        response,
			}).
			Error(err.Error())

		return nil, fmt.Errorf("internal server error")

	}

	prof.Status = 1

	return prof, nil

}

func CreditWalletProfile(tr trace.Tracer, ctx context.Context, client Client, credit Credit) (*CreditTransactionResponse, error) {

	ctx, span := tr.Start(ctx, "CreditWalletProfile")
	defer span.End()

	spanID := span.SpanContext().SpanID().String()
	traceID := span.SpanContext().TraceID().String()

	providerID, _ := strconv.ParseInt(os.Getenv("PROVIDER_ID"), 10, 64)
	providerName := os.Getenv("PROVIDER_NAME")

	headers := map[string]string{
		client.AuthenticationHeader: client.AuthenticationString,
		"span-id":                   spanID,
		"trace-id":                  traceID,
	}

	creditRequest := CreditRequest{
		PlayerID:           credit.PlayerID,
		ProviderID:         providerID,
		ProviderName:       providerName,
		GameName:           credit.GameName,
		GameID:             credit.GameID,
		TransactionID:      credit.TransactionID,
		Amount:             credit.Amount,
		SessionID:          credit.SessionID,
		RoundID:            credit.RoundID,
		SpanID:             spanID,
		TraceID:            traceID,
		DebitTransactionID: credit.DebitTransactionID,
		FreeSpinWin:        credit.FreeSpinWin,
	}

	endpoint := fmt.Sprintf("%s/credit", client.BaseURL)

	status, response := HTTPPost(ctx, endpoint, headers, creditRequest)

	if status > 299 || status < 200 {

		logrus.WithContext(ctx).
			WithFields(logrus.Fields{
				"description":      fmt.Sprintf("invalid status %d crediting user ", status),
				"endpoint":         endpoint,
				"request":          creditRequest,
				"response_status":  status,
				"response_payload": response,
			}).
			Warn(response)

		if status == http.StatusConflict {

			prof := new(CreditTransactionResponse)
			prof.Status = http.StatusConflict
			prof.Description = response
			return prof, nil
		}

		return nil, fmt.Errorf("%s", response)

	} else {

		logrus.WithContext(ctx).
			WithFields(logrus.Fields{
				"endpoint":         endpoint,
				"request":          creditRequest,
				"response_status":  status,
				"response_payload": response,
			}).
			Info(response)
	}

	prof := new(CreditTransactionResponse)

	err := json.Unmarshal([]byte(response), prof)
	if err != nil {

		logrus.WithContext(ctx).
			WithFields(logrus.Fields{
				"description": "error unmarshalling TransactionResponse from JSON",
				"data":        response,
			}).
			Error(err.Error())

		return nil, fmt.Errorf("internal server error")

	}

	prof.Status = 1
	return prof, nil

}

func BetSettlement(tr trace.Tracer, ctx context.Context, client Client, settlement Settlement) error {

	ctx, span := tr.Start(ctx, "BetSettlement")
	defer span.End()

	spanID := span.SpanContext().SpanID().String()
	traceID := span.SpanContext().TraceID().String()

	providerID, _ := strconv.ParseInt(os.Getenv("PROVIDER_ID"), 10, 64)

	headers := map[string]string{
		client.AuthenticationHeader: client.AuthenticationString,
		"span-id":                   spanID,
		"trace-id":                  traceID,
	}

	settlementRequest := SettlementRequest{
		PlayerID:           settlement.PlayerID,
		Status:             settlement.Status,
		SessionID:          settlement.SessionID,
		RoundID:            settlement.RoundID,
		DebitTransactionID: settlement.DebitTransactionID,
		ProviderID:         providerID,
	}

	endpoint := fmt.Sprintf("%s/settlement", client.BaseURL)

	status, response := HTTPPost(ctx, endpoint, headers, settlementRequest)
	if status > 299 || status < 200 {

		logrus.WithContext(ctx).
			WithFields(logrus.Fields{
				"description":      fmt.Sprintf("invalid status %d on bet settlement ", status),
				"endpoint":         endpoint,
				"request":          settlementRequest,
				"response_status":  status,
				"response_payload": response,
			}).
			Warn(response)

		return fmt.Errorf("%s", response)
	}

	return nil

}

func AdjustWalletProfile(tr trace.Tracer, ctx context.Context, client Client, adjustment Adjustment) (*AdjustmentTransactionResponse, error) {

	ctx, span := tr.Start(ctx, "AdjustWalletProfile")
	defer span.End()

	spanID := span.SpanContext().SpanID().String()
	traceID := span.SpanContext().TraceID().String()

	providerID, _ := strconv.ParseInt(os.Getenv("PROVIDER_ID"), 10, 64)
	providerName := os.Getenv("PROVIDER_NAME")

	headers := map[string]string{
		client.AuthenticationHeader: client.AuthenticationString,
		"span-id":                   spanID,
		"trace-id":                  traceID,
	}

	adjustmentRequest := AdjustmentRequest{
		ProviderID:    providerID,
		ProviderName:  providerName,
		PlayerID:      adjustment.PlayerID,
		GameName:      adjustment.GameName,
		GameID:        adjustment.GameID,
		TransactionID: adjustment.TransactionID,
		Amount:        adjustment.Amount,
		SessionID:     adjustment.SessionID,
		RoundID:       adjustment.RoundID,
		FreeSpinWin:   adjustment.FreeSpinWin,
	}

	endpoint := fmt.Sprintf("%s/adjust", client.BaseURL)

	status, response := HTTPPost(ctx, endpoint, headers, adjustmentRequest)

	if status > 299 || status < 200 {

		logrus.WithContext(ctx).
			WithFields(logrus.Fields{
				"description":      fmt.Sprintf("invalid status %d crediting user ", status),
				"endpoint":         endpoint,
				"request":          adjustmentRequest,
				"response_status":  status,
				"response_payload": response,
			}).
			Warn(response)

		if status == http.StatusConflict {

			prof := new(AdjustmentTransactionResponse)
			prof.Status = http.StatusConflict
			prof.Description = response
			return prof, nil
		}

		return nil, fmt.Errorf("%s", response)

	} else {

		logrus.WithContext(ctx).
			WithFields(logrus.Fields{
				"endpoint":         endpoint,
				"request":          adjustmentRequest,
				"response_status":  status,
				"response_payload": response,
			}).
			Info(response)
	}

	prof := new(AdjustmentTransactionResponse)

	err := json.Unmarshal([]byte(response), prof)
	if err != nil {

		logrus.WithContext(ctx).
			WithFields(logrus.Fields{
				"description": "error unmarshalling TransactionResponse from JSON",
				"data":        response,
			}).
			Error(err.Error())

		return nil, fmt.Errorf("internal server error")

	}

	prof.Status = 1
	return prof, nil

}

func BetRollback(tr trace.Tracer, ctx context.Context, client Client, rollback Rollback) (*RollbackTransactionResponse, error) {

	ctx, span := tr.Start(ctx, "BetRollback")
	defer span.End()

	spanID := span.SpanContext().SpanID().String()
	traceID := span.SpanContext().TraceID().String()

	providerID, _ := strconv.ParseInt(os.Getenv("PROVIDER_ID"), 10, 64)
	providerName := os.Getenv("PROVIDER_NAME")

	headers := map[string]string{
		client.AuthenticationHeader: client.AuthenticationString,
		"span-id":                   spanID,
		"trace-id":                  traceID,
	}

	rollbackRequest := RollbackRequest{
		ProviderID:         providerID,
		ProviderName:       providerName,
		PlayerID:           rollback.PlayerID,
		TransactionID:      rollback.TransactionID,
		Amount:             rollback.Amount,
		SessionID:          rollback.SessionID,
		RoundID:            rollback.RoundID,
		DebitTransactionID: rollback.DebitTransactionID,
	}

	endpoint := fmt.Sprintf("%s/rollback", client.BaseURL)

	status, response := HTTPPost(ctx, endpoint, headers, rollbackRequest)

	if status > 299 || status < 200 {

		logrus.WithContext(ctx).
			WithFields(logrus.Fields{
				"description":      fmt.Sprintf("invalid status %d on bet rollback ", status),
				"endpoint":         endpoint,
				"request":          rollbackRequest,
				"response_status":  status,
				"response_payload": response,
			}).
			Warn(response)

		if status == http.StatusConflict {

			prof := new(RollbackTransactionResponse)
			prof.Status = http.StatusConflict
			prof.Description = response
			return prof, nil
		}

		return nil, fmt.Errorf("%s", response)

	} else {

		logrus.WithContext(ctx).
			WithFields(logrus.Fields{
				"endpoint":         endpoint,
				"request":          rollbackRequest,
				"response_status":  status,
				"response_payload": response,
			}).
			Info(response)
	}

	prof := new(RollbackTransactionResponse)

	err := json.Unmarshal([]byte(response), prof)
	if err != nil {

		logrus.WithContext(ctx).
			WithFields(logrus.Fields{
				"description": "error unmarshalling TransactionResponse from JSON",
				"data":        response,
			}).
			Error(err.Error())

		return nil, fmt.Errorf("internal server error")

	}

	prof.Status = 1
	return prof, nil

}

func HTTPPost(ctx context.Context, url string, headers map[string]string, payload interface{}) (httpStatus int, response string) {

	if payload == nil {

		payload = "{}"
	}

	jsonData, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {

		logrus.WithContext(ctx).
			WithFields(logrus.Fields{
				"description": "got error making http request",
				"endpoint":    url,
				"request":     payload,
			}).
			Error(err.Error())

		return 0, ""
	}

	ua := fmt.Sprintf("Touchvas Gaming/2.5 (provider;%s) (providerID;%s)", os.Getenv("PROVIDER_NAME"), os.Getenv("PROVIDER_ID"))

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", ua)

	if headers != nil {

		for k, v := range headers {

			req.Header.Set(k, v)
		}
	}

	resp, err := NewNetClient().Do(req)
	if err != nil {

		logrus.WithContext(ctx).
			WithFields(logrus.Fields{
				"description": "got error making http request",
				"endpoint":    url,
				"request":     payload,
			}).
			Error(err.Error())

		return 0, ""
	}

	st := resp.StatusCode
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {

		logrus.WithContext(ctx).
			WithFields(logrus.Fields{
				"description":     "got error making http request",
				"endpoint":        url,
				"request":         payload,
				"response_status": st,
			}).
			Error(err.Error())

		return st, ""
	}

	return st, string(body)
}

func NewNetClient() *http.Client {

	once.Do(func() {

		var netTransport = &http.Transport{
			Dial: (&net.Dialer{
				Timeout: 30 * time.Second,
			}).Dial,
			DialContext: (&net.Dialer{
				Timeout: 30 * time.Second,
			}).DialContext,
			TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
			TLSHandshakeTimeout: 30 * time.Second,
		}

		netClient = &http.Client{
			Timeout:   time.Second * 30,
			Transport: otelhttp.NewTransport(netTransport),
		}
	})

	return netClient
}
