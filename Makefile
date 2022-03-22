p := txpay
t := txpay
lang:
	easyi18n generate --pkg=locales ./locales ./locales/locales.go

api:
	goctl api go -api $(p)/$(t).api -dir $(p) -remote https://github.com/neccohuang/go-zero-template

tx:
	go run $(p)/$(t).go -f $(p)/etc/$(t).yaml -env $(p)/etc/.env
