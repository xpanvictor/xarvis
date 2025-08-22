package conversation

type Message struct {
	Id     string   `json:"id"`
	UserId string   `json:"user_id"`
	Text   string   `json:"text"`
	Tags   []string `json:"tags"`
}

// Single conversation per user
type ConversationRepository interface {
	CreateMessage(userId string, msg Message) (Message, error)
	FetchUserMessages(userId string) ([]Message, error)
	FetchMessage(msgId string) (Message, error)
}
