package wallet

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
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

func GetWalletProfile(tr trace.Tracer, ctx context.Context, client Client, profileID string, decimalMultiplier DecimalMultiplier) (*WalletProfile, error) {

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

	if client.APIVersion > 0 {

		endpoint = fmt.Sprintf("%s/v%d/profile", client.BaseURL, client.APIVersion)

	}

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

	if client.APIVersion > 0 {

		// 3400 0000
		// 100 (3400 00)
		// balance correction
		if decimalMultiplier.In64() != DecimalMultiplierTenOfThousands {

			prof.Balance = int64(prof.Balance * decimalMultiplier.In64() / DecimalMultiplierTenOfThousands)

		}
	}

	return prof, nil

}

func DebitWalletProfile(tr trace.Tracer, ctx context.Context, client Client, debit Debit, decimalMultiplier DecimalMultiplier) (*DebitTransactionResponse, error) {

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

	if client.APIVersion > 0 {

		endpoint = fmt.Sprintf("%s/v%d/debit", client.BaseURL, client.APIVersion)

		if decimalMultiplier.In64() != DecimalMultiplierTenOfThousands {

			debitRequest.Amount = int64(debitRequest.Amount * DecimalMultiplierTenOfThousands / decimalMultiplier.In64())

		}
	}

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
	if client.APIVersion > 0 {

		// balance correction
		if decimalMultiplier.In64() != DecimalMultiplierTenOfThousands {

			prof.Balance = int64(prof.Balance * decimalMultiplier.In64() / DecimalMultiplierTenOfThousands)

		}
	}

	return prof, nil

}

func CreditWalletProfile(tr trace.Tracer, ctx context.Context, client Client, credit Credit, decimalMultiplier DecimalMultiplier) (*CreditTransactionResponse, error) {

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

	if client.APIVersion > 0 {

		endpoint = fmt.Sprintf("%s/v%d/credit", client.BaseURL, client.APIVersion)

		if decimalMultiplier.In64() != DecimalMultiplierTenOfThousands {

			creditRequest.Amount = int64(creditRequest.Amount * DecimalMultiplierTenOfThousands / decimalMultiplier.In64())

		}
	}

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

	if client.APIVersion > 0 {

		// balance correction
		if decimalMultiplier.In64() != DecimalMultiplierTenOfThousands {

			prof.Balance = int64(prof.Balance * decimalMultiplier.In64() / DecimalMultiplierTenOfThousands)

		}
	}

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

	if client.APIVersion > 0 {

		endpoint = fmt.Sprintf("%s/v%d/settlement", client.BaseURL, client.APIVersion)

	}

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

func AdjustWalletProfile(tr trace.Tracer, ctx context.Context, client Client, adjustment Adjustment, decimalMultiplier DecimalMultiplier) (*AdjustmentTransactionResponse, error) {

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

	if client.APIVersion > 0 {

		endpoint = fmt.Sprintf("%s/v%d/adjust", client.BaseURL, client.APIVersion)

		if decimalMultiplier.In64() != DecimalMultiplierTenOfThousands {

			adjustmentRequest.Amount = int64(adjustmentRequest.Amount * DecimalMultiplierTenOfThousands / decimalMultiplier.In64())

		}
	}

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

	if client.APIVersion > 0 {

		// balance correction
		if decimalMultiplier.In64() != DecimalMultiplierTenOfThousands {

			prof.Balance = int64(prof.Balance * decimalMultiplier.In64() / DecimalMultiplierTenOfThousands)

		}
	}

	return prof, nil

}

func BetRollback(tr trace.Tracer, ctx context.Context, client Client, rollback Rollback, decimalMultiplier DecimalMultiplier) (*RollbackTransactionResponse, error) {

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

	if client.APIVersion > 0 {

		endpoint = fmt.Sprintf("%s/v%d/rollback", client.BaseURL, client.APIVersion)

		if decimalMultiplier.In64() != DecimalMultiplierTenOfThousands {

			rollbackRequest.Amount = int64(rollbackRequest.Amount * DecimalMultiplierTenOfThousands / decimalMultiplier.In64())

		}
	}

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

	if client.APIVersion > 0 {

		// balance correction
		if decimalMultiplier.In64() != DecimalMultiplierTenOfThousands {

			prof.Balance = int64(prof.Balance * decimalMultiplier.In64() / DecimalMultiplierTenOfThousands)

		}
	}

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
