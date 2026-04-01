package gemini

import "gemini-wrapper/model"

// GeminiStatus captures upstream status metadata returned by Gemini requests.

type GeminiService interface {
	Ask(question string, model string) (string, *model.GeminiStatus, error)
	AskWithEnv(question string, model string, _ map[string]string) (string, *model.GeminiStatus, error)
}
