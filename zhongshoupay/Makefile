t := zhongshoupay
lang:
	easyi18n generate --pkg=locales ./locales ./locales/locales.go

api:
	goctl api go -api $(t).api -dir . -remote https://github.com/neccohuang/go-zero-template

run:
	go run $(t).go -f etc/$(t).yaml -env etc/.env
