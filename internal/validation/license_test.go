package validation

import "testing"

func TestLicenseStatus(t *testing.T) {
	cases := map[string]string{
		"":            LicenseUnknown,
		"NOASSERTION": LicenseUnknown,
		"MIT":         LicenseValid,
		"mit":         LicenseValid,
		"Apache-2.0":  LicenseValid,
		"GPL-3.0":     LicenseValid,
		"proprietary": LicenseUnsupported,
	}
	for in, want := range cases {
		if got := LicenseStatus(in); got != want {
			t.Fatalf("LicenseStatus(%q)=%s want %s", in, got, want)
		}
	}
}

func TestRecognizedLicenses(t *testing.T) {
	ids := []string{"MIT", "Apache-2.0", "GPL-2.0", "GPL-3.0", "LGPL-2.1", "LGPL-3.0", "BSD-2-Clause", "BSD-3-Clause", "MPL-2.0", "ISC", "EPL-2.0", "AGPL-3.0"}
	for _, id := range ids {
		if !IsRecognizedLicense(id) {
			t.Fatalf("%s should be recognized", id)
		}
	}
}

func TestUnknownIsNotVerifiedOpenSource(t *testing.T) {
	if HasOpenSourceLicense("") || HasOpenSourceLicense("NOASSERTION") {
		t.Fatal("missing license must not count as open source")
	}
}
