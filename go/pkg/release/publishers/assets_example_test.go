package publishers

import core "dappco.re/go"

func ExampleChecksumMap() {
	checksums := ChecksumMap{
		LinuxAmd64File: "app_linux_amd64.tar.gz",
		LinuxAmd64:     "sha256",
	}
	core.Println(checksums.LinuxAmd64File, checksums.LinuxAmd64)
	// Output: app_linux_amd64.tar.gz sha256
}
