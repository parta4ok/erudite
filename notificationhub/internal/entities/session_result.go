package entities

type SessionResult struct {
	userID     string
	Topics     []string
	Questions  map[int]string
	UserAnswer map[string][]string
	IsExpire   bool
	IsSuccess  bool
	Resume     string
	Recipient  *Recipient
}

type Recipient struct {
	ID       string
	Contacts map[string]string
}

func (sr *SessionResult) GetUserID() string {
	return sr.userID
}

func (sr *SessionResult) SetRecipient(recipient *Recipient) {
	sr.Recipient = recipient
}
