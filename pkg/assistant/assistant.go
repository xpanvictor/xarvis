package assistant

func NewAssistantInput(
	msgs []AssistantMessage,
	meta interface{},
) AssistantInput {
	return AssistantInput{
		Msgs:           msgs,
		AvailableTools: make([]AssistantToolType, 0),
		Meta:           nil,
	}
}
