package utils

import "testing"

func TestParseReadableSize(t *testing.T) {
	tt := []struct {
		Input  string
		Expect ReadableSize
	}{
		{"0", ReadableSize(0 * B)},
		{"0", ReadableSize(0 * KIB)},
		{"0B", ReadableSize(0 * B)},
		{"0B", ReadableSize(0 * KIB)},
		{"0KB", ReadableSize(0 * B)},
		{"0MB", ReadableSize(0 * KIB)},
		{"10MB", ReadableSize(10 * MIB)},
		{"10M", ReadableSize(10 * MIB)},
		{"10MiB", ReadableSize(10 * MIB)},
		{"10KB", ReadableSize(10 * KIB)},
		{"10K", ReadableSize(10 * KIB)},
		{"10KiB", ReadableSize(10 * KIB)},
		{"10GB", ReadableSize(10 * GIB)},
		{"10G", ReadableSize(10 * GIB)},
		{"10GiB", ReadableSize(10 * GIB)},
		{"0.5GB", ReadableSize(512 * MIB)},
		{"0.1KB", ReadableSize(102 * B)},
		{"1GB", ReadableSize(1024 * MIB)},
	}

	for _, tc := range tt {
		t.Run("", func(t *testing.T) {
			rSize, err := ParseReadableSize(tc.Input)
			if err != nil {
				t.Fatal(err)
			}
			if rSize != tc.Expect {
				t.Fatalf("%v != %v", rSize, tc.Expect)
			}
		})
	}
}

func TestMustCmpReadableSize(t *testing.T) {
	tt := []struct {
		Left   string
		Right  string
		Expect int
	}{
		{"0GB", "0", 0},
		{"0GB", "0MB", 0},
		{"0", "1B", -1},
		{"1GB", "1TB", -1},
		{"1GB", "2GB", -1},
		{"1GB", "1G", 0},
		{"2048MB", "1GB", 1},
		{"1MB", "2048KB", -1},
		{"1MB", "1024KB", 0},
	}

	for _, tc := range tt {
		t.Run("", func(t *testing.T) {
			res := MustCmpReadableSize(tc.Left, tc.Right)
			if res != tc.Expect {
				t.Errorf("%+v no match %+v", res, tc.Expect)
			}
		})
	}
}

func TestMustCmpDuration(t *testing.T) {
	tt := []struct {
		Left   string
		Right  string
		Expect int
	}{
		{"0us", "0ms", 0},
		{"0s", "0h", 0},
		{"0", "0m", 0},
		{"60s", "1m", 0},
		{"60m", "1h", 0},
		{"0", "1m", -1},
		{"1s", "2s", -1},
		{"1h", "1d", -1},
		{"61s", "1m", 1},
		{"61m", "1h", 1},
		{"1m", "61s", -1},
		{"1h", "61m", -1},
		{"1us", "1m", -1},
		{"1us", "1ms", -1},
		{"1h1m", "1h2m", -1},
		{"1m10s", "2m", -1},
	}

	for _, tc := range tt {
		t.Run("", func(t *testing.T) {
			res := MustCmpDuration(tc.Left, tc.Right)
			if res != tc.Expect {
				t.Errorf("%+v no match %+v", res, tc.Expect)
			}
		})
	}
}

func TestParseReadableDuration(t *testing.T) {
	durStr := "0s"
	_, err := ParseReadableDuration(durStr)
	if err != nil {
		t.Fatal(err)
	}

}
