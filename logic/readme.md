The `logic` directory.

This directory contains all buissness code. The `logic` interface connects the `frontend` with the `database` and provides logic which gets provided to the `frontend` for display

```markdown
logic/
├── admin.go          // functions specific to the admin pages
├── config.go         // provides constants and data handling functions directly accessing the `database`
├── cycles.go         // functions specific to the watch cycles
├── dataimporter.go   // functions specific to the used apis to autofill movie submissions
├── link.go           // functions specificly operating on/with `link` structs
├── logic.go          // provides the `logic` interface and the `backend` implementation aswell as some general functions
├── movies.go         // functions specifically operating on/with `movie` structures
├── readme.md
├── security.go       // functions used for passwords/encryption/keys etc
├── user.go           // functions specifically operating on/with `user` structures
└── vote.go           // functions specifically operating on/with `vote` structures
```
