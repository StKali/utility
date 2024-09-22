PROGRAM=utility
REMOVE=@rm -f
PRINT=@echo
GO=@go

COVER_FILE=cover.out
COVER_HTML=coverage.html
TEMPORARY=temporary

all: test clean

test: setup mock
	$(GO) test -v ./... -coverprofile=$(COVER_FILE)
	$(PRINT) "Start render coverage report to $(COVER_HTML)."
	$(GO) tool cover --html=cover.out -o $(COVER_HTML)
	$(PRINT) "create coverage report at: $(COVER_HTML)."

mock:
	$(GO) generate ./...
	$(PRINT) "Successfully generate mock files."

clean:
	$(REMOVE) $(COVER_FILE) $(COVER_HTML) *.log *.out *.prof *.test *.test.log main.go
	 $(REMOVE) -r $(TEMPORARY)
	$(PRINT) "Clean up done"

setup:
	$(PRINT) "Installing dependencies..."
	$(GO) mod tidy
	$(GO) mod download
	$(PRINT) "Successfully installed dependencies."
	$(GO) install go.uber.org/mock/mockgen@latest
	$(PRINT) "Successfully installed mockgen."
	$(PRINT) "Successfully installed go toolchain."


.PHONY: test mock clean env setup
