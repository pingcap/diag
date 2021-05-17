package version

import (
	"fmt"

	tiupver "github.com/pingcap/tiup/pkg/version"
)

var (
	ReleaseVersion string = ""
	GitBranch      string = ""
	GitHash        string = ""
)

func PrintReleaseInfo() {
	fmt.Println("Release Version:", ReleaseVersion)
	fmt.Println("Git Branch:", GitBranch)
	fmt.Println("Git Commit Hash:", GitHash)
	fmt.Println("TiUP Version:", tiupver.NewTiUPVersion().SemVer())
}

func String() string {
	return fmt.Sprintf("%s @%s (%s) feat. tiup v%s", ReleaseVersion, GitBranch, GitHash, tiupver.NewTiUPVersion().SemVer())
}
