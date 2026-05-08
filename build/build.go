package build

const Token = "cmw" // Token is arbitrary short tag for this service; it carries no semantic meaning.

var serviceName, version, shortName string // Injected at build time - check Makefile.

func ServiceName() string { return serviceName }
func Version() string     { return version }

func init() {
	if Dev {
		version += "+dev"
	}
}
