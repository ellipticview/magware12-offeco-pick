package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHTTPOrdersClientFetchReadyForPicking(t *testing.T) {
	t.Run("successful pagination", func(t *testing.T) {
		var requestCount int
		client := NewHTTPOrdersClient("https://example.invalid/api/v1/orders", &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				requestCount++
				require.Equal(t, "Bearer startup-token", r.Header.Get("Authorization"))
				require.Equal(t, "status", r.URL.Query().Get("searchAttribute"))
				require.Equal(t, "=", r.URL.Query().Get("searchDelimiter"))
				require.Equal(t, "ready_for_picking", r.URL.Query().Get("searchValue"))
				require.Equal(t, "createDate", r.URL.Query().Get("orderBy"))
				require.Equal(t, "asc", r.URL.Query().Get("orderByDirection"))
				require.Equal(t, "50", r.URL.Query().Get("perPage"))

				switch r.URL.Query().Get("page") {
				case "1":
					return responseWithBody(http.StatusOK, `{
						"pageInfo":{"lastPage":2},
						"items":[
							{"externalDisplayId":"ORD-1","status":"ready_for_picking","shippingCountry":"NL"},
							{"external_display_id":"ORD-2","status":"ready_for_picking","shipping_country":"BE"}
						]
					}`)
				case "2":
					return responseWithBody(http.StatusOK, `{
						"pageInfo":{"lastPage":2},
						"items":[
							{"externalDisplayId":"ORD-3","status":"cancelled","shippingCountry":"NL"}
						]
					}`)
				default:
					return responseWithBody(http.StatusNotFound, `{"error":"not found"}`)
				}
			}),
		})

		var items []RemoteOrderData
		var sequences []int

		err := client.FetchReadyForPicking(context.Background(), "startup-token", func(order RemoteOrderData, sequence int) error {
			items = append(items, order)
			sequences = append(sequences, sequence)
			return nil
		})

		require.NoError(t, err)
		require.Equal(t, 2, requestCount)
		require.Len(t, items, 3)
		require.Equal(t, []int{1, 2, 3}, sequences)
		require.Equal(t, "ORD-1", items[0].ExternalDisplayID)
		require.Equal(t, "BE", normalizedCountry(items[1].ShippingCountry))
	})

	t.Run("non success response aborts", func(t *testing.T) {
		client := NewHTTPOrdersClient("https://example.invalid/api/v1/orders", &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				return responseWithBody(http.StatusBadGateway, "kapot\n")
			}),
		})

		err := client.FetchReadyForPicking(context.Background(), "startup-token", func(order RemoteOrderData, sequence int) error {
			return nil
		})

		require.Error(t, err)
		require.Contains(t, err.Error(), "502")
		require.Contains(t, err.Error(), "kapot")
	})

	t.Run("invalid json aborts", func(t *testing.T) {
		client := NewHTTPOrdersClient("https://example.invalid/api/v1/orders", &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				return responseWithBody(http.StatusOK, `{`)
			}),
		})

		err := client.FetchReadyForPicking(context.Background(), "startup-token", func(order RemoteOrderData, sequence int) error {
			return nil
		})

		require.Error(t, err)
		require.Contains(t, err.Error(), "ongeldige JSON")
	})

	t.Run("missing last page aborts", func(t *testing.T) {
		client := NewHTTPOrdersClient("https://example.invalid/api/v1/orders", &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				return responseWithBody(http.StatusOK, `{"pageInfo":{},"items":[]}`)
			}),
		})

		err := client.FetchReadyForPicking(context.Background(), "startup-token", func(order RemoteOrderData, sequence int) error {
			return nil
		})

		require.Error(t, err)
		require.Contains(t, err.Error(), "lastPage")
	})

	t.Run("missing items aborts", func(t *testing.T) {
		client := NewHTTPOrdersClient("https://example.invalid/api/v1/orders", &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				return responseWithBody(http.StatusOK, `{"pageInfo":{"lastPage":1}}`)
			}),
		})

		err := client.FetchReadyForPicking(context.Background(), "startup-token", func(order RemoteOrderData, sequence int) error {
			return nil
		})

		require.Error(t, err)
		require.Contains(t, err.Error(), "items")
	})

	t.Run("later page failure aborts after earlier success", func(t *testing.T) {
		var callbackCount int
		client := NewHTTPOrdersClient("https://example.invalid/api/v1/orders", &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				switch r.URL.Query().Get("page") {
				case "1":
					return responseWithBody(http.StatusOK, `{
						"pageInfo":{"lastPage":2},
						"items":[{"externalDisplayId":"ORD-1","status":"ready_for_picking","shippingCountry":"NL"}]
					}`)
				case "2":
					return responseWithBody(http.StatusInternalServerError, "pagina 2 stuk\n")
				default:
					return responseWithBody(http.StatusNotFound, `{"error":"not found"}`)
				}
			}),
		})

		err := client.FetchReadyForPicking(context.Background(), "startup-token", func(order RemoteOrderData, sequence int) error {
			callbackCount++
			return nil
		})

		require.Error(t, err)
		require.Equal(t, 1, callbackCount)
		require.Contains(t, err.Error(), "pagina 2 stuk")
	})
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func responseWithBody(statusCode int, body string) (*http.Response, error) {
	return &http.Response{
		StatusCode: statusCode,
		Status:     fmt.Sprintf("%d %s", statusCode, http.StatusText(statusCode)),
		Header: http.Header{
			"Content-Type": []string{"application/json"},
		},
		Body: io.NopCloser(strings.NewReader(body)),
	}, nil
}
