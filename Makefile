PROGRAM		=	utility
REMOVE		=	@rm -f
PRINT		=	@echo
GO			=	@go
COVER_FILE	=	cover.out
COVER_HTML	=	coverage.html
TEMPORARY	=	temporary
SUPPORT_GO_VERSIONS = 1.18 1.19 1.20 1.21 1.22 1.23

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
	$(GO) install go.uber.org/mock/mockgen@v0.4.0
	$(PRINT) "successfully installed mockgen."
	$(PRINT) "successfully installed go toolchain."

compatibility_test:
	@for version in $(SUPPORT_GO_VERSIONS); do \
  docker run --rm -v $(PWD):/app -e GOPROXY=$(go env GOPROXY) -d golang:$$version sh cd /app && make test; \
  echo "successfully test for go $$version."; \
  done \


.PHONY: test mock clean env setup
