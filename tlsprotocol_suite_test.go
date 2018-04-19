package tlsprotocol

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestTlsprotocol(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Tlsprotocol Suite")
}
