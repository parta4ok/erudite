package public

type Introspector interface {
	Introspect(userID, jwt string) error
}
