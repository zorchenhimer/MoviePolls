EXE=
ifeq ($(OS), Windows_NT)
EXE=.exe
endif

SOURCES = \
		  logic/admin.go \
		  logic/auth.go \
		  models/authmethod.go \
		  models/cycle.go \
		  models/logger.go \
		  models/movie.go \
		  models/user.go \
		  models/util.go \
		  models/vote.go \
		  models/link.go \
		  data/connector.go \
		  data/json.go \
		  data/mysql.go \
		  logic/dataimporter.go \
		  server/server.go \
		  logic/session.go \
		  templates/templates.go \
		  logic/user.go \
		  logic/util.go \
		  logic/oauth.go \
		  logic/votes.go

.PHONY: all data fmt server

CMD_SERVER=bin/server$(EXE)
CMD_DATA=bin/mkdata$(EXE)


RELEASEVERSION ?=$(shell git describe --tags --dirty --broken)

all: fmt $(CMD_SERVER)
data: fmt $(CMD_DATA)

server: main.go fmt $(SOURCES)
	GOOS=linux GOARCH=386 go$(GO_VERSION) build -ldflags "-X github.com/zorchenhimer/MoviePolls.ReleaseVersion=${RELEASEVERSION}" -o bin/MoviePolls $<

clean:
	rm -f $(CMD_SERVER) $(CMD_DATA) bin/MoviePolls

fmt:
	gofmt -w .

$(CMD_SERVER): main.go $(SOURCES)
	go$(GO_VERSION) build -ldflags "-X github.com/zorchenhimer/MoviePolls.ReleaseVersion=${RELEASEVERSION}" -o $@ $<

$(CMD_DATA): scripts/mkdata.go $(SOURCES)
	go$(GO_VERSION) build -ldflags "-X github.com/zorchenhimer/MoviePolls.ReleaseVersion=${RELEASEVERSION}" -o $@ $<

run: all
	cmd/server
