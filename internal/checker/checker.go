package checker

import "context"

type Result struct {
	Address   string
	Online    bool
	LatencyMS int
	Attempts  int
	TimeoutMS int
}
type Checker interface {
	Ping(context.Context, string, int, int) Result
}
