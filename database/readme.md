The `database` directory.

This directory contains all database specific code. The backend communicates directly with the database via the `DatabaseConnector`.

``` markdown
database/
├── database.go       // defines the `DatabaseConnector` interface
├── database_test.go  // tests for the `DatabaseConnector` interface
├── helpers_test.go
├── json.go           // JSON implmentation of the `DatabaseConnector`
├── mysql             // directory contining a **REALLY** old db dump
├── mysql.go          // MySQL implmentation of the `DatabaseConnector`
└── readme.md
```
