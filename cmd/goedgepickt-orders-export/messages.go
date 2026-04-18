package main

import (
	"fmt"
	"strings"
	"time"
)

const distributionDateLayout = "2006-01-02"

func tokenValidationErrorMessage(reason error) string {
	return fmt.Sprintf("Tokenvalidatie mislukt: %v", reason)
}

func exportProgressMessage(country string, city string) string {
	trimmedCity := strings.TrimSpace(city)
	if trimmedCity == "" {
		return fmt.Sprintf("Bestelling voor %s geëxporteerd.", country)
	}

	return fmt.Sprintf("Bestelling voor %s (%s) geëxporteerd.", trimmedCity, country)
}

func completionSummaryMessage(nlCount int, beCount int, totalCount int) string {
	return fmt.Sprintf("Download voltooid: %d NL, %d BE, %d totaal.", nlCount, beCount, totalCount)
}

func downloadErrorMessage(reason error) string {
	return fmt.Sprintf("Download mislukt: %v", reason)
}

func distributionDate(now time.Time) string {
	return now.Format(distributionDateLayout)
}

func exportName(firstName string, lastName string, externalDisplayID string) string {
	return strings.TrimSpace(strings.Join([]string{
		strings.TrimSpace(firstName),
		strings.TrimSpace(lastName),
		strings.TrimSpace(externalDisplayID),
	}, " "))
}

func orderedProgressText(messages []ProgressMessage) string {
	lines := make([]string, 0, len(messages))
	for _, message := range messages {
		lines = append(lines, message.Text)
	}

	return strings.Join(lines, "\n")
}
