package public

import "context"

type Introspector interface {
	Introspect(ctx context.Context, userID, jwt string) error
}
