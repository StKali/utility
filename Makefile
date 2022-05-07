test:
	go test -v ./... -coverprofile=cover.out
	echo "start render coverage report to coverage.html"
	go tool cover --html=cover.out -o coverage.html
	echo "create coverage report at: coverage.html"
