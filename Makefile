
SOURCES = api.go \
		  data.go \
		  data_interfaces.go \
		  data_json.go \
		  server.go \
		  templates.go

.PHONY: all data

all: cmd/server
data: cmd/mkdata

clean:
	rm -f cmd/mkdata cmd/server

cmd/server: cmd/server.go $(SOURCES)
	go build -o cmd/server cmd/server.go

cmd/mkdata: cmd/mkdata.go $(SOURCES)
	go build -o cmd/mkdata cmd/mkdata.go

run: all
	cmd/server