package command

import (
	"github.com/onsi/gomega"
	"testing"
)

func Test_ParseURL(t *testing.T) {
	opt := &downloadOptions{}
	err := parseURL(opt, "url")

	g := gomega.NewWithT(t)
	g.Expect(err).To(gomega.MatchError("invalid url"))

	err = parseURL(opt, "")
	g.Expect(err).To(gomega.MatchError("invalid url"))

	err = parseURL(opt, " ")
	g.Expect(err).To(gomega.MatchError("invalid url"))

	err = parseURL(opt, "https://clinic.com/api/files/uuid")
	g.Expect(err).To(gomega.Succeed())

	g.Expect(opt.endpoint).To(gomega.Equal("https://clinic.com"))
	g.Expect(opt.fileUUID).To(gomega.Equal("uuid"))

}
