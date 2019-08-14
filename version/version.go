package version

import (
	"fmt"
)

var (
	ReleaseVersion string = ""
	GitBranch      string = ""
	GitHash        string = ""
	BuildTS        string = ""
)

func PrintReleaseInfo() {
	fmt.Println("Release Version:", ReleaseVersion)
	fmt.Println("Git Branch:", GitBranch)
	fmt.Println("Git Commit Hash:", GitHash)
	fmt.Println("UTC Build Time:", BuildTS)
}
