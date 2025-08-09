package cases

type Notifier interface {
	Next() (Notifier, error)
	
}
