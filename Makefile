PROGRAM=utility
REMOVE=@rm -f

# cover
COVERFILE=cover.out
COVERHTML=coverage.html

test:
	go test -v ./... -coverprofile=$(COVERFILE)
	echo "start render coverage report to $(COVERHTML)"
	go tool cover --html=cover.out -o $(COVERHTML)
	echo "create coverage report at: $(COVERHTML)"

clean:
	$(REMOVE) $(COVERFILE) $(COVERHTML) main.go

