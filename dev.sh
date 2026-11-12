#!/usr/bin/env bash
# Live reload for both programs; Ctrl+C stops both.
air --build.cmd "go build -o tmp/dispatcher.exe ./cmd/dispatcher" --build.entrypoint tmp/dispatcher.exe &
air --build.cmd "go build -o tmp/router.exe ./cmd/router" --build.entrypoint tmp/router.exe &
trap 'kill 0' EXIT
wait
