package main

import "strings"

func stringPtrValue(value *string) string {
	if value == nil {
		return ""
	}

	return *value
}

func normalizedCountry(value *string) string {
	return strings.ToUpper(strings.TrimSpace(stringPtrValue(value)))
}

func exportHouseNumberAddition(addition *string, address2 *string) string {
	parts := []string{}

	for _, part := range []string{
		strings.TrimSpace(stringPtrValue(addition)),
		strings.TrimSpace(stringPtrValue(address2)),
	} {
		if part != "" {
			parts = append(parts, part)
		}
	}

	return strings.Join(parts, " ")
}

func isQualifyingOrder(order RemoteOrderData) bool {
	if order.Status != "ready_for_picking" {
		return false
	}

	switch normalizedCountry(order.ShippingCountry) {
	case "NL", "BE":
		return true
	default:
		return false
	}
}

func mapCSVRow(order RemoteOrderData, now string) CsvRow {
	return CsvRow{
		ID:                   order.ExternalDisplayID,
		DatumDistributiedag:  now,
		Naam:                 exportName(stringPtrValue(order.ShippingFirstName), stringPtrValue(order.ShippingLastName), order.ExternalDisplayID),
		Straatnaam:           stringPtrValue(order.BillingAddress),
		Huisnummer:           stringPtrValue(order.ShippingHouseNumber),
		Huisnummertoevoeging: exportHouseNumberAddition(order.ShippingHouseNumberAddition, order.ShippingAddress2),
		Postcode:             stringPtrValue(order.ShippingZipcode),
		Plaatsnaam:           stringPtrValue(order.ShippingCity),
		CountryCode:          normalizedCountry(order.ShippingCountry),
		TelefoonNummer:       stringPtrValue(order.BillingPhone),
		Email:                stringPtrValue(order.BillingEmail),
		BezoekenNa:           "",
		BezoekenVoor:         "",
		Locatieinstructie:    stringPtrValue(order.CustomerNote),
	}
}
