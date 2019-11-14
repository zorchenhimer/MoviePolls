
SOURCES = api.go \
		  data.go \
		  data_interfaces.go \
		  data_json.go \
		  server.go \
		  templates.go

cmd/server: cmd/main.go $(SOURCES)
	cd cmd && go build -o server

