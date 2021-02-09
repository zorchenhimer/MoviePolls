EXE=
ifeq ($(OS), Windows_NT)
EXE=.exe
endif

SOURCES = \
		  admin.go \
		  api.go \
		  auth.go \
		  common/authmethod.go \
		  common/cycle.go \
		  common/logger.go \
		  common/movie.go \
		  common/user.go \
		  common/util.go \
		  common/vote.go \
		  common/link.go \
		  data/connector.go \
		  data/json.go \
		  data/mysql.go \
		  dataimporter.go \
		  server.go \
		  session.go \
		  templates.go \
		  user.go \
		  util.go \
		  oauth.go \
		  votes.go

.PHONY: all data fmt server

CMD_SERVER=bin/server$(EXE)
CMD_DATA=bin/mkdata$(EXE)


RELEASEVERSION ?=$(shell git describe --tags --abbrev=0 | head -n 1)
# just casually replacing all space characters with non breaking space characters because of reasons...
COMMITHASH ?=$(shell git log -1 --pretty='format:%h: %s - %an' | sed 's/ / /g')

all: fmt $(CMD_SERVER)
data: fmt $(CMD_DATA)

server: cmd/server.go fmt $(SOURCES)
	GOOS=linux GOARCH=386 go$(GO_VERSION) build -ldflags "-X github.com/zorchenhimer/MoviePolls.ReleaseVersion=${RELEASEVERSION} -X github.com/zorchenhimer/MoviePolls.Commit=${COMMITHASH}" -o bin/MoviePolls $<

clean:
	rm -f $(CMD_SERVER) $(CMD_DATA) bin/MoviePolls

fmt:
	gofmt -w .

$(CMD_SERVER): cmd/server.go $(SOURCES)
	echo $(COMMITHASH)
	go$(GO_VERSION) build -ldflags '-X github.com/zorchenhimer/MoviePolls.ReleaseVersion=${RELEASEVERSION} -X github.com/zorchenhimer/MoviePolls.Commit=${COMMITHASH}' -o $@ $<

$(CMD_DATA): cmd/mkdata.go $(SOURCES)
	go$(GO_VERSION) build -ldflags "-X github.com/zorchenhimer/MoviePolls.ReleaseVersion=${RELEASEVERSION} -X github.com/zorchenhimer/MoviePolls.Commit=${COMMITHASH}" -o $@ $<

run: all
	cmd/server
