EXE=
ifeq ($(OS), Windows_NT)
EXE=.exe
endif

SOURCES = \
		  database/database.go\
		  database/json.go\
		  logic/config.go\
		  logic/cycles.go\
		  logic/movies.go\
		  logic/security.go\
		  logic/user.go\
		  logic/vote.go\
		  logic/vote.go\
		  main.go\
		  models/authmethod.go\
		  models/cycle.go\
		  models/error.go\
		  models/link.go\
		  models/logger.go\
		  models/movie.go\
		  models/tag.go\
		  models/urlkey.go\
		  models/user.go\
		  models/util.go\
		  models/vote.go\
		  web/handlersAuth.go\
		  web/handlerStatic.go\
		  web/pageAddMovie.go\
		  web/pageHistory.go\
		  web/pageMain.go\
		  web/pageMovie.go\
		  web/pageUser.go\
		  web/server.go\
		  web/session.go\
		  web/templates.go\
		  web/template_structs.go
		  
.PHONY: all data fmt server

CMD_SERVER=bin/server$(EXE)
CMD_DATA=bin/mkdata$(EXE)


RELEASEVERSION ?=$(shell git describe --tags --dirty --broken)

all: fmt $(CMD_SERVER)
data: fmt $(CMD_DATA)

server: main.go fmt $(SOURCES)
	GOOS=linux GOARCH=386 go$(GO_VERSION) build -ldflags "-X main.ReleaseVersion=${RELEASEVERSION}" -o bin/MoviePolls $<

clean:
	rm -f $(CMD_SERVER) $(CMD_DATA) bin/MoviePolls

fmt:
	@echo "gofmt -w {SOURCES}" && gofmt -w $(SOURCES) 

$(CMD_SERVER): main.go $(SOURCES)
	go$(GO_VERSION) build -ldflags "-X main.ReleaseVersion=${RELEASEVERSION}" -o $@ $<

$(CMD_DATA): scripts/mkdata.go $(SOURCES)
	go$(GO_VERSION) build -ldflags "-X main.ReleaseVersion=${RELEASEVERSION}" -o $@ $<

run: all
	cmd/server
