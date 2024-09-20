PROGRAM=utility
REMOVE=@rm -f
PRINT=@echo
GO=@go

COVER_FILE=cover.out
COVER_HTML=coverage.html
TEMPORARY=temporary

all: test clean

test: env mock
	$(GO) test -v ./... -coverprofile=$(COVER_FILE)
	$(PRINT) "Start render coverage report to $(COVER_HTML)."
	$(GO) tool cover --html=cover.out -o $(COVER_HTML)
	$(PRINT) "create coverage report at: $(COVER_HTML)."

mock:
	$(GO) generate ./...
	$(PRINT) "Successfully generate mock files."

clean:
	$(REMOVE) $(COVER_FILE) $(COVER_HTML) $(TEMPORARY) *.log *.out *.prof *.test *.test.log main.go
	$(PRINT) "Clean up done"

env:
	$(PRINT) "Initializing environment..."
	$(GO) mod download
	$(PRINT) "Successfully downloaded dependencies."
	@./scripts/setup_env.sh
	$(PRINT) "Successfully installed build toolchain."

.PHONY: test mock clean env
