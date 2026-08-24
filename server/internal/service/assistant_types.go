package service

import "time"

type AssistantValue[T any] struct {
	Value      *T         `json:"value"`
	Display    string     `json:"display"`
	Reason     string     `json:"reason,omitempty"`
	Provider   string     `json:"provider,omitempty"`
	SourceURL  string     `json:"sourceUrl,omitempty"`
	ObservedAt *time.Time `json:"observedAt,omitempty"`
	Freshness  string     `json:"freshness"`
}

func knownValue[T any](value T, display, provider, source string, observed time.Time, freshness string) AssistantValue[T] {
	return AssistantValue[T]{Value: &value, Display: display, Provider: provider, SourceURL: source, ObservedAt: &observed, Freshness: freshness}
}
func unknownValue[T any](reason string) AssistantValue[T] {
	return AssistantValue[T]{Display: "--", Reason: reason, Freshness: "unavailable"}
}
