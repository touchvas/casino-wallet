package wallet

type Client struct {
	ID                   int64
	BaseURL              string
	AuthenticationString string
	AuthenticationHeader string
}

type TransactionResponse struct {
	Status        int64   `json:"status"`
	BetID         int64   `json:"bet_id"`
	Balance       float64 `json:"balance"`
	TransactionID int64   `json:"transaction_id"`
	Description   string  `json:"description"`
}

type DebitTransactionResponse struct {
	BonusBet      int64   `json:"bonus_bet"`
	BonusBalance  float64 `json:"bonus_balance"`
	Balance       float64 `json:"balance"`
	BonusDeducted float64 `json:"bonus_deducted"`
	Status        int64   `json:"status"`
	Description   string  `json:"description"`
}

type CreditTransactionResponse struct {
	BonusBalance float64 `json:"bonus_balance"`
	Balance      float64 `json:"balance"`
	Status       int64   `json:"status"`
	Description  string  `json:"description"`
}

type RollbackTransactionResponse struct {
	BonusBalance float64 `json:"bonus_balance"`
	Balance      float64 `json:"balance"`
	Status       int64   `json:"status"`
	Description  string  `json:"description"`
}

type AdjustmentTransactionResponse struct {
	BonusBalance float64 `json:"bonus_balance"`
	Balance      float64 `json:"balance"`
	Status       int64   `json:"status"`
	Description  string  `json:"description"`
}

type WalletProfile struct {
	DisplayName string  `json:"display_name"`
	ID          string  `json:"player_id"`
	Balance     float64 `json:"balance"`
	Bonus       float64 `json:"bonus"`
}

type Debit struct {
	PlayerID      string  `json:"player_id"`
	GameName      string  `json:"game_name"`
	GameID        string  `json:"game_id"`
	TransactionID string  `json:"transaction_id"`
	Amount        float64 `json:"amount"`
	SessionID     string  `json:"session_id"`
	RoundID       string  `json:"round_id"`
}

type Credit struct {
	PlayerID           string  `json:"player_id"`
	GameName           string  `json:"game_name"`
	GameID             string  `json:"game_id"`
	TransactionID      string  `json:"transaction_id"`
	DebitTransactionID string  `json:"debit_transaction_id"`
	Amount             float64 `json:"amount"`
	SessionID          string  `json:"session_id"`
	RoundID            string  `json:"round_id"`
}

type Adjustment struct {
	PlayerID      string  `json:"player_id"`
	GameName      string  `json:"game_name"`
	GameID        string  `json:"game_id"`
	TransactionID string  `json:"transaction_id"`
	Amount        float64 `json:"amount"`
	SessionID     string  `json:"session_id"`
	RoundID       string  `json:"round_id"`
}

type Rollback struct {
	PlayerID           string  `json:"player_id"`
	TransactionID      string  `json:"transaction_id"`
	Amount             float64 `json:"amount"`
	SessionID          string  `json:"session_id"`
	RoundID            string  `json:"round_id"`
	DebitTransactionID string  `json:"debit_transaction_id"`
}

type DebitRequest struct {
	PlayerID      string  `json:"player_id"`
	ProviderID    int64   `json:"provider_id"`
	ProviderName  string  `json:"provider_name"`
	GameName      string  `json:"game_name"`
	GameID        string  `json:"game_id"`
	TransactionID string  `json:"transaction_id"`
	Amount        float64 `json:"amount"`
	SessionID     string  `json:"session_id"`
	RoundID       string  `json:"round_id"`
	SpanID        string  `json:"span_id"`
	TraceID       string  `json:"trace_id"`
}

type CreditRequest struct {
	PlayerID           string  `json:"player_id"`
	ProviderID         int64   `json:"provider_id"`
	ProviderName       string  `json:"provider_name"`
	GameName           string  `json:"game_name"`
	GameID             string  `json:"game_id"`
	TransactionID      string  `json:"transaction_id"`
	Amount             float64 `json:"amount"`
	SessionID          string  `json:"session_id"`
	RoundID            string  `json:"round_id"`
	SpanID             string  `json:"span_id"`
	TraceID            string  `json:"trace_id"`
	DebitTransactionID string  `json:"debit_transaction_id"`
}

type AdjustmentRequest struct {
	ProviderID    int64   `json:"provider_id"`
	ProviderName  string  `json:"provider_name"`
	PlayerID      string  `json:"player_id"`
	GameName      string  `json:"game_name"`
	GameID        string  `json:"game_id"`
	TransactionID string  `json:"transaction_id"`
	Amount        float64 `json:"amount"`
	SessionID     string  `json:"session_id"`
	RoundID       string  `json:"round_id"`
}

type ProfileRequest struct {
	PlayerID string `json:"player_id"`
	SpanID   string `json:"span_id"`
	TraceID  string `json:"trace_id"`
}

type RollbackRequest struct {
	ProviderID         int64   `json:"provider_id"`
	ProviderName       string  `json:"provider_name"`
	PlayerID           string  `json:"player_id"`
	TransactionID      string  `json:"transaction_id"`
	Amount             float64 `json:"amount"`
	SessionID          string  `json:"session_id"`
	RoundID            string  `json:"round_id"`
	DebitTransactionID string  `json:"debit_transaction_id"`
}

type Settlement struct {
	PlayerID           string `json:"player_id"`
	Status             int64  `json:"status"`
	SessionID          string `json:"session_id"`
	RoundID            string `json:"round_id"`
	DebitTransactionID string `json:"debit_transaction_id"`
}

type SettlementRequest struct {
	ProviderID         int64  `json:"provider_id"`
	PlayerID           string `json:"player_id"`
	Status             int64  `json:"status"`
	SessionID          string `json:"session_id"`
	RoundID            string `json:"round_id"`
	DebitTransactionID string `json:"debit_transaction_id"`
}