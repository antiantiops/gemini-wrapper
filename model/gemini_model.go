package model

type AskRequest struct {
	Question string `json:"question" validate:"required"`
	Model    string `json:"model,omitempty"`
}

type AskResponse struct {
	Answer string        `json:"answer"`
	Error  string        `json:"error,omitempty"`
	Status *GeminiStatus `json:"status,omitempty"`
}

type GeminiAPIRequest struct {
	Contents []struct {
		Parts []struct {
			Text string `json:"text"`
		} `json:"parts"`
	} `json:"contents"`
}

type GeminiAPIResponse struct {
	Model      string `json:"model"`
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
	Status *GeminiStatus `json:"status,omitempty"`
}

// For Gemini Service internal use

type GeminiStatus struct {
	HTTPStatus int    `json:"httpStatus"`
	Code       string `json:"code,omitempty"`
	Message    string `json:"message,omitempty"`
}
