test-coverage:
	mkdir -p coverage
	go test -covermode=count -coverpkg=. -coverprofile coverage/cover.out .
	go tool cover -html coverage/cover.out -o coverage/cover.html
	rm coverage/cover.out
