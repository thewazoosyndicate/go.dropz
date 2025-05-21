package common

// UpdateNotifier is an interface for components that need to be notified of updates
type UpdateNotifier interface {
	// NotifyUpdate notifies the implementing component that an update occurred
	NotifyUpdate()
}
