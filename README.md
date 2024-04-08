# routine pool - Golang routine pool for multi asynchronous tasks.

[![GoDoc](https://godoc.org/github.com/icefed/routinepool?status.svg)](https://pkg.go.dev/github.com/icefed/routinepool)
[![Go Report Card](https://goreportcard.com/badge/github.com/icefed/routinepool)](https://goreportcard.com/report/github.com/icefed/routinepool)
![Build Status](https://github.com/icefed/routinepool/actions/workflows/test.yml/badge.svg)
[![Coverage](https://img.shields.io/codecov/c/github/icefed/routinepool)](https://codecov.io/gh/icefed/routinepool)
[![License](https://img.shields.io/github/license/icefed/routinepool)](./LICENSE)

### Supports
- Automatic scaling Number of routines by the number of tasks.
- Support waiting all tasks finished.
- Panic in tasks will be recovered.
