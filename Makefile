PROGRAM		=	utility
REMOVE		=	@rm -f
PRINT		=	@echo
GO			=	@go
COVER_FILE	=	cover.out
COVER_HTML	=	coverage.html
TEMPORARY	=	temporary

all: test clean

test: setup mock
	$(GO) test -v ./... -coverprofile=$(COVER_FILE)
	$(PRINT) "start render coverage report to $(COVER_HTML)."
	$(GO) tool cover --html=$(COVER_FILE) -o $(COVER_HTML)
	$(PRINT) "create coverage report at: $(COVER_HTML)."

mock:
	$(GO) generate ./...
	$(PRINT) "successfully generate mock files."

clean:
	$(REMOVE) $(COVER_FILE) $(COVER_HTML) *.log *.out *.prof *.test *.test.log main.go
	$(REMOVE) -r $(TEMPORARY)
	$(PRINT) "clean up done"

setup:
	$(PRINT) "installing dependencies..."
	$(GO) mod tidy
	$(GO) mod download
	$(PRINT) "successfully installed dependencies."
	$(GO) install go.uber.org/mock/mockgen@latest
	$(PRINT) "successfully installed mockgen."
	$(PRINT) "successfully installed go toolchain."

.PHONY: test mock clean env setup
