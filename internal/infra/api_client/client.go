package api_client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/NastyaGoryachaya/crypto-rate-service/internal/config"
	"github.com/NastyaGoryachaya/crypto-rate-service/internal/domain"
)

type Client struct {
	cfg        config.CoinGeckoConfig
	httpClient *http.Client
}

// coingeckoResponse — структура для парсинга ответа API CoinGecko
type coingeckoResponse struct {
	ID           string  `json:"id"`
	Symbol       string  `json:"symbol"`
	CurrentPrice float64 `json:"current_price"`
}

// NewClient - Создаёт нового клиента для работы с API CoinGecko.
func NewClient(cfg config.CoinGeckoConfig) *Client {
	return &Client{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
}

// FetchRates — получает курсы валют по API CoinGecko
func (c *Client) FetchRates(ctx context.Context) ([]domain.Coin, error) {
	u, err := url.Parse(c.cfg.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}
	u.Path, _ = url.JoinPath(u.Path, "coins", "markets")

	q := u.Query()
	q.Set("vs_currency", strings.ToLower(c.cfg.Currency))
	q.Set("ids", strings.Join(c.cfg.Coins, ","))
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	ua := c.cfg.UserAgent
	if ua == "" {
		ua = "crypto-rate-service/1.0 (+https://github.com/NastyaGoryachaya/crypto-rate-service)"
	}
	req.Header.Set("User-Agent", ua)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed: %s", resp.Status)
	}

	var data []coingeckoResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	var result []domain.Coin
	for _, d := range data {
		result = append(result, domain.Coin{
			Symbol:    strings.ToUpper(d.Symbol),
			Price:     d.CurrentPrice,
			UpdatedAt: time.Now().UTC(),
		})
	}
	return result, nil
}
