// Package meta wraps Meta (Facebook) Graph API calls relevant to Commerce
// Catalog. It implements checkout.CatalogClient, so consumers depend on the
// domain interface rather than this concrete type.
//
// Spec verified via Context7 + live testing against catalog 1017309634048260:
//   - Endpoint: GET /{version}/{catalog_id}/products
//   - Filter: ?filter={"retailer_id":{"is_any":[ids]}}
//   - Auth: Authorization: Bearer <token> (header, NOT query param —
//     keeps token out of paginated next/previous URLs returned by Meta)
//   - Response price format: localized "Rp460.000" (IDR string)
//   - Currency: ISO 4217 (we accept IDR only)
//
// HTTP transport uses Fiber's fasthttp-based client for connection-pool reuse.
// JSON decode uses bytedance/sonic for consistency with the rest of the service.
package meta

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v3/client"

	"github.com/blastcoid/dhianstore/services/checkout-url/internal/checkout"
	"github.com/blastcoid/dhianstore/services/checkout-url/internal/config"
)

const (
	// DefaultTimeout caps each Graph API request.
	DefaultTimeout = 10 * time.Second

	// expectedCurrency is hard-coded — multi-currency is out of scope. A
	// product with a different currency in the catalog signals merchant
	// misconfiguration and we surface it as a CatalogError, not a silent
	// fallback (would risk wrong gross_amount in Midtrans).
	expectedCurrency = "IDR"
)

// Client fetches products from the configured Meta Catalog.
type Client struct {
	httpClient  *client.Client
	endpointURL string
	authHeader  string
}

// Error wraps Graph API failures and catalog-shape problems (currency
// mismatch, price parse error). StatusCode is 0 for non-HTTP failures.
type Error struct {
	Message      string
	StatusCode   int
	ResponseBody string
	Cause        error
}

func (e *Error) Error() string {
	if e.StatusCode != 0 {
		return fmt.Sprintf("meta: %s (status=%d)", e.Message, e.StatusCode)
	}
	return fmt.Sprintf("meta: %s", e.Message)
}

func (e *Error) Unwrap() error { return e.Cause }

// New constructs a client. Tests substitute the upstream URL by overriding
// cfg.MetaGraphAPIBase (e.g., pointing at httptest.NewServer).
func New(cfg *config.Config) *Client {
	base := strings.TrimRight(cfg.MetaGraphAPIBase, "/")
	endpoint := fmt.Sprintf("%s/%s/%s/products", base, cfg.MetaGraphVersion, cfg.MetaCatalogID)
	return &Client{
		httpClient:  client.New().SetTimeout(DefaultTimeout),
		endpointURL: endpoint,
		authHeader:  "Bearer " + cfg.MetaAccessToken,
	}
}

// graphResponse mirrors only the shape we read; extra fields ignored.
type graphResponse struct {
	Data []graphProduct `json:"data"`
}

type graphProduct struct {
	RetailerID string `json:"retailer_id"`
	Name       string `json:"name"`
	Price      string `json:"price"`
	Currency   string `json:"currency"`
}

// FetchProducts implements checkout.CatalogClient. Single bulk request via
// is_any filter — efficient for typical 1-5 item carts. Returns:
//   - *checkout.ProductNotFoundError if any requested retailerID is absent
//     from the catalog response (first missing one is reported).
//   - *Error for transport / response shape / parse failures.
func (c *Client) FetchProducts(ctx context.Context, retailerIDs []string) ([]checkout.Product, error) {
	if len(retailerIDs) == 0 {
		return nil, &Error{Message: "no retailer IDs to fetch"}
	}

	filter, err := sonic.MarshalString(map[string]any{
		"retailer_id": map[string]any{"is_any": retailerIDs},
	})
	if err != nil {
		return nil, &Error{Message: "marshal filter", Cause: err}
	}

	resp, err := c.httpClient.R().
		SetContext(ctx).
		SetHeader("Authorization", c.authHeader).
		SetHeader("Accept", "application/json").
		SetParam("filter", filter).
		SetParam("fields", "retailer_id,name,price,currency").
		Get(c.endpointURL)
	if err != nil {
		return nil, &Error{Message: "network: " + err.Error(), Cause: err}
	}
	defer resp.Close()

	status := resp.StatusCode()
	body := resp.Body()

	if status < 200 || status >= 300 {
		return nil, &Error{
			Message:      fmt.Sprintf("graph api rejected request: %d", status),
			StatusCode:   status,
			ResponseBody: string(body),
		}
	}

	var parsed graphResponse
	if err := sonic.Unmarshal(body, &parsed); err != nil {
		return nil, &Error{
			Message:      "non-JSON success body",
			StatusCode:   status,
			ResponseBody: string(body),
			Cause:        err,
		}
	}

	// Index by retailer_id for O(1) lookup and missing detection.
	byID := make(map[string]graphProduct, len(parsed.Data))
	for _, p := range parsed.Data {
		byID[p.RetailerID] = p
	}

	out := make([]checkout.Product, 0, len(retailerIDs))
	for _, id := range retailerIDs {
		gp, ok := byID[id]
		if !ok {
			return nil, &checkout.ProductNotFoundError{ProductID: id}
		}
		if gp.Currency != expectedCurrency {
			return nil, &Error{
				Message: fmt.Sprintf(
					"product %q has unexpected currency %q (want %q)",
					id, gp.Currency, expectedCurrency,
				),
			}
		}
		price, err := checkout.ParseIDRPrice(gp.Price)
		if err != nil {
			return nil, &Error{
				Message: fmt.Sprintf("product %q price parse: %v", id, err),
				Cause:   err,
			}
		}
		out = append(out, checkout.Product{
			ID:    gp.RetailerID,
			Name:  gp.Name,
			Price: price,
		})
	}
	return out, nil
}
