package notifier

type Notifier interface {
	TestWebhook() error
	SendStatusChange(checkName, url string, isUp bool, statusCode int, responseTimeMs int, errorMsg string) error
}

