package public

import "context"

type Introspector interface {
	Introspect(ctx context.Context, jwt string) error
}
