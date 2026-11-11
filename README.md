# pointsman

![architecture](./architecture.svg)

Pointsman is a router that runs the loop of many AI agents in parallel.\
It calls the model, records each session and delegates tool execution to external sandboxes.\
Since tool execution is decoupled from the harness, the router only passes data between the sandboxes and the model. A single router can therefore handle many simultaneous sessions.


## Spec
See [SPEC.md](SPEC.md) for what this project is and how it is designed.

## Run
Copy `.env.example` to `.env` update the values.

`.env` is read unless `ENVIRONMENT=production`.

Run the dispatcher: `go run ./cmd/dispatcher`.

Run the router: `go run ./cmd/router`.

## Test
Build: `go build ./...`. Test: `go test ./...`.
