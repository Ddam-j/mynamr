package version

import "fmt"

var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

func Detailed() string {
	return fmt.Sprintf("mynamr %s (commit: %s, built at: %s)", Version, Commit, Date)
}
