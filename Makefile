EXE=
ifeq ($(OS), Windows_NT)
EXE=.exe
endif

SOURCES = \
		  admin.go \
		  api.go \
		  common/cycle.go \
		  common/movie.go \
		  common/user.go \
		  common/util.go \
		  common/vote.go \
		  data/connector.go \
		  data/json.go \
		  data/mysql.go \
		  server.go \
		  session.go \
		  templates.go

.PHONY: all data fmt server

CMD_SERVER=cmd/server$(EXE)
CMD_DATA=cmd/mkdata$(EXE)

all: fmt $(CMD_SERVER) server
data: fmt $(CMD_DATA)

server: cmd/server.go fmt $(SOURCES)
	GOOS=linux GOARCH=386 go build -o cmd/MoviePolls $<

clean:
	rm -f $(CMD_SERVER) $(CMD_DATA) cmd/MoviePolls

fmt:
	gofmt -w .

$(CMD_SERVER): cmd/server.go $(SOURCES)
	go build -o $@ $<

$(CMD_DATA): cmd/mkdata.go $(SOURCES)
	go build -o $@ $<

run: all
	cmd/server
