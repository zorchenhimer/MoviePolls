EXE=
ifeq ($(OS), Windows_NT)
EXE=.exe
endif

SOURCES = api.go \
		  data.go \
		  data_interfaces.go \
		  data_json.go \
		  server.go \
		  templates.go

.PHONY: all data fmt

CMD_SERVER=cmd/server$(EXE)
CMD_DATA=cmd/mkdata$(EXE)

all: fmt $(CMD_SERVER)
data: fmt $(CMD_DATA)

clean:
	rm -f $(CMD_SERVER) $(CMD_DATA)

fmt:
	gofmt -w .

$(CMD_SERVER): cmd/server.go $(SOURCES)
	go build -o $@ $<

$(CMD_DATA): cmd/mkdata.go $(SOURCES)
	go build -o $@ $<

run: all
	cmd/server