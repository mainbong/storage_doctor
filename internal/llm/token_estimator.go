package llm

import (
	"math"
	"unicode/utf8"
)

func EstimateTokens(messages []Message) int {
	total := 0
	for _, msg := range messages {
		total += estimateTextTokens(msg.Content) + 4
	}
	if total < 1 {
		return 1
	}
	return total
}

func estimateTextTokens(text string) int {
	if text == "" {
		return 0
	}
	bytes := len(text)
	runes := utf8.RuneCountInString(text)
	estimate := int(math.Ceil(float64(bytes) / 3.0))
	if estimate < runes/2 {
		estimate = runes / 2
	}
	if estimate < 1 {
		estimate = 1
	}
	return estimate
}
