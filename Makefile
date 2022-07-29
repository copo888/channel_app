p := shenfutongpay6
t := shenfutongpay6
lang:
	easyi18n generate --pkg=locales ./locales ./locales/locales.go

api:
	goctl api go -api $(p)/pay.api -dir $(p) -remote https://github.com/neccohuang/go-zero-template

tx:
	go run $(p)/$(t).go -f $(p)/etc/$(t).yaml -env $(p)/etc/.env
