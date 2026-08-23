default:
  just --list

[no-exit-message]
@webserver *args:
  go run ./cmd/webserver {{args}}
