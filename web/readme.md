The `web` directory.

This directory contains everything related to the web frontend.

``` markdown
web/
├── handlersAuth.go       // contains the handlers used for (O)auth
├── handlerStatic.go      // contains the handlers for serving static files (contained inside the `static` folder)
├── pageAddMovie.go       // contains the handlers for the `/add/` route
├── pageAdmin.go          // contains the handlers for the `/admin/` route
├── pageHistory.go        // contains the handlers for the `/history/` route
├── pageMain.go           // contains the handlers for the `/` route
├── pageMovie.go          // contains the handlers for the `/movie/` route
├── pageUser.go           // contains the handlers for the `/user/` route
├── readme.md
├── server.go             // contains the `webServer` struct definitions, assigns the handlers to the routes etc.
├── session.go            // contains all the session logic
├── static/               // contains all static files as well as css
├── templates/            // contains all the html template files
├── templates.go          // contains code used for templating
└── template_structs.go   // contains all data structs used for templating
```
