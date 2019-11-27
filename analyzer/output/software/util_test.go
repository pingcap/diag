package software

import "testing"

func TestLoadVersionsTag(t *testing.T) {
	version := SoftwareVersion{version: "001", os: "macos"}
	versions := make([]SoftwareVersion, 0)
	versions = append(versions, version)

	v1 := loadVersionsTag(versions, "version")
	if len(v1) != 1 {
		t.Fail()
	}
	if v1[0] != "001" {
		t.Fail()
	}

	versions = append(versions, SoftwareVersion{version: "002", os: "windows"})

	v2 := loadVersionsTag(versions, "os")
	if len(v2) != 2 {
		t.Fail()
	}
	if v2[0] != "macos" && v2[1] != "windows" {
		t.Fail()
	}
}
