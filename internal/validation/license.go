package validation

import "strings"

const (
	LicenseValid       = "valid"
	LicenseUnknown     = "unknown"
	LicenseUnsupported = "unsupported"
)

var recognizedLicenses = map[string]struct{}{
	"MIT":               {},
	"Apache-2.0":        {},
	"GPL-2.0":           {},
	"GPL-2.0-only":      {},
	"GPL-2.0-or-later":  {},
	"GPL-3.0":           {},
	"GPL-3.0-only":      {},
	"GPL-3.0-or-later":  {},
	"LGPL-2.1":          {},
	"LGPL-2.1-only":     {},
	"LGPL-2.1-or-later": {},
	"LGPL-3.0":          {},
	"LGPL-3.0-only":     {},
	"LGPL-3.0-or-later": {},
	"BSD-2-Clause":      {},
	"BSD-3-Clause":      {},
	"MPL-2.0":           {},
	"ISC":               {},
	"EPL-2.0":           {},
	"AGPL-3.0":          {},
	"AGPL-3.0-only":     {},
	"AGPL-3.0-or-later": {},
	"Unlicense":         {},
	"0BSD":              {},
	"BSL-1.0":           {},
	"CC0-1.0":           {},
	"Artistic-2.0":      {},
	"Zlib":              {},
}

var licenseAliases = map[string]string{
	"mit":          "MIT",
	"apache-2.0":   "Apache-2.0",
	"apache 2.0":   "Apache-2.0",
	"gpl-2.0":      "GPL-2.0",
	"gpl-3.0":      "GPL-3.0",
	"lgpl-2.1":     "LGPL-2.1",
	"lgpl-3.0":     "LGPL-3.0",
	"bsd-2-clause": "BSD-2-Clause",
	"bsd-3-clause": "BSD-3-Clause",
	"mpl-2.0":      "MPL-2.0",
	"isc":          "ISC",
	"epl-2.0":      "EPL-2.0",
	"agpl-3.0":     "AGPL-3.0",
	"other":        "",
	"noassertion":  "",
	"none":         "",
}

func NormalizeLicense(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return ""
	}
	if _, ok := recognizedLicenses[id]; ok {
		return id
	}
	lower := strings.ToLower(id)
	if alias, ok := licenseAliases[lower]; ok {
		return alias
	}
	for known := range recognizedLicenses {
		if strings.EqualFold(known, id) {
			return known
		}
	}
	return id
}

func LicenseStatus(id string) string {
	id = NormalizeLicense(id)
	if id == "" {
		return LicenseUnknown
	}
	if _, ok := recognizedLicenses[id]; ok {
		return LicenseValid
	}
	return LicenseUnsupported
}

func IsRecognizedLicense(id string) bool {
	return LicenseStatus(id) == LicenseValid
}

func HasOpenSourceLicense(id string) bool {
	return IsRecognizedLicense(id)
}
