package operations

import (
	"github.com/tympanix/nexus-cli/internal/checksum"
	"github.com/tympanix/nexus-cli/internal/util"
)

func processKeyTemplateWrapper(input string, keyFromFile string) (string, error) {
	return util.ProcessKeyTemplate(input, keyFromFile, checksum.ComputeChecksum)
}
