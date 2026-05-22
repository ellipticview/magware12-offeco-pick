package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type OrdersClient interface {
	FetchReadyForPicking(ctx context.Context, token string, onOrder func(RemoteOrderData, int) error) error
}

type HTTPOrdersClient struct {
	baseURL string
	client  *http.Client
}

func NewHTTPOrdersClient(baseURL string, client *http.Client) *HTTPOrdersClient {
	if client == nil {
		client = http.DefaultClient
	}

	return &HTTPOrdersClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  client,
	}
}

func (c *HTTPOrdersClient) FetchReadyForPicking(ctx context.Context, token string, onOrder func(RemoteOrderData, int) error) error {
	sequenceNumber := 0
	lastPage := 1

	for page := 1; page <= lastPage; page++ {
		items, resolvedLastPage, err := c.fetchPage(ctx, token, page)
		if err != nil {
			return err
		}
		lastPage = resolvedLastPage

		for _, item := range items {
			sequenceNumber++
			if err := onOrder(item, sequenceNumber); err != nil {
				return err
			}
		}
	}

	return nil
}

func (c *HTTPOrdersClient) fetchPage(ctx context.Context, token string, page int) ([]RemoteOrderData, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("kan orders-aanvraag niet maken: %w", err)
	}

	values := url.Values{}
	values.Set("page", strconv.Itoa(page))
	values.Set("searchAttribute", "status")
	values.Set("searchDelimiter", "=")
	values.Set("searchValue", "ready_for_picking")
	values.Set("orderBy", "createDate")
	values.Set("orderByDirection", "asc")
	values.Set("perPage", "50")
	req.URL.RawQuery = values.Encode()
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("orders-aanvraag mislukt: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, fmt.Errorf("kan orders-response niet lezen: %w", err)
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, 0, fmt.Errorf("orders-api gaf status %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	var payload map[string]json.RawMessage
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, 0, fmt.Errorf("ongeldige JSON van orders-api: %w", err)
	}

	lastPage, err := extractLastPage(payload)
	if err != nil {
		return nil, 0, err
	}

	items, err := extractItems(payload)
	if err != nil {
		return nil, 0, err
	}

	return items, lastPage, nil
}

func extractLastPage(payload map[string]json.RawMessage) (int, error) {
	var pageInfo map[string]json.RawMessage
	pageInfoRaw, ok := payload["pageInfo"]
	if !ok {
		return 0, fmt.Errorf("orders-response mist pageInfo.lastPage")
	}

	if err := json.Unmarshal(pageInfoRaw, &pageInfo); err != nil {
		return 0, fmt.Errorf("orders-response bevat ongeldige pageInfo: %w", err)
	}

	lastPageRaw, ok := pageInfo["lastPage"]
	if !ok {
		return 0, fmt.Errorf("orders-response mist pageInfo.lastPage")
	}

	var lastPage int
	if err := json.Unmarshal(lastPageRaw, &lastPage); err != nil {
		return 0, fmt.Errorf("orders-response bevat een ongeldige pageInfo.lastPage: %w", err)
	}

	if lastPage < 1 {
		return 0, fmt.Errorf("orders-response bevat een ongeldige pageInfo.lastPage: %d", lastPage)
	}

	return lastPage, nil
}

func extractItems(payload map[string]json.RawMessage) ([]RemoteOrderData, error) {
	itemsRaw, ok := payload["items"]
	if !ok {
		return nil, fmt.Errorf("orders-response mist items")
	}
	if string(itemsRaw) == "null" {
		return nil, fmt.Errorf("orders-response mist items")
	}

	var rawItems []map[string]json.RawMessage
	if err := json.Unmarshal(itemsRaw, &rawItems); err != nil {
		return nil, fmt.Errorf("orders-response bevat ongeldige items: %w", err)
	}

	items := make([]RemoteOrderData, 0, len(rawItems))
	for _, rawItem := range rawItems {
		items = append(items, decodeRemoteOrder(rawItem))
	}

	return items, nil
}

func decodeRemoteOrder(raw map[string]json.RawMessage) RemoteOrderData {
	return RemoteOrderData{
		ExternalDisplayID:           readRequiredString(raw, "externalDisplayId"),
		Status:                      readRequiredString(raw, "status"),
		BillingPhone:                readOptionalString(raw, "billingPhone"),
		BillingEmail:                readOptionalString(raw, "billingEmail"),
		ShippingFirstName:           readOptionalString(raw, "shippingFirstName"),
		ShippingLastName:            readOptionalString(raw, "shippingLastName"),
		ShippingAddress:             readOptionalString(raw, "shippingAddress"),
		ShippingHouseNumber:         readOptionalString(raw, "shippingHouseNumber"),
		ShippingHouseNumberAddition: readOptionalString(raw, "shippingHousenumberAddition"),
		ShippingAddress2:            readOptionalString(raw, "shippingAddress2"),
		ShippingZipcode:             readOptionalString(raw, "shippingZipcode"),
		ShippingCity:                readOptionalString(raw, "shippingCity"),
		ShippingCountry:             readOptionalString(raw, "shippingCountry"),
		CustomerNote:                readOptionalString(raw, "customerNote"),
	}
}

func readRequiredString(raw map[string]json.RawMessage, key string) string {
	valueRaw, ok := raw[key]
	if !ok {
		return ""
	}

	var value string
	if err := json.Unmarshal(valueRaw, &value); err == nil {
		return value
	}
	return ""
}

func readOptionalString(raw map[string]json.RawMessage, key string) *string {
	valueRaw, ok := raw[key]
	if !ok {
		return nil
	}

	if string(valueRaw) == "null" {
		return nil
	}

	var value string
	if err := json.Unmarshal(valueRaw, &value); err == nil {
		return &value
	}
	return nil
}
