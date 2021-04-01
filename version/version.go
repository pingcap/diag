package version

import (
	"fmt"
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
}

func String() string {
	return fmt.Sprintf("%s @%s (%s)", ReleaseVersion, GitBranch, GitHash)
}
