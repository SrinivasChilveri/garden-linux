package lifecycle_test

import (
	"fmt"
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Garden startup flags", func() {

	var debugAddr string

	BeforeEach(func() {
		debugAddr = fmt.Sprintf("0.0.0.0:%d", 15000+GinkgoParallelNode())
	})

	Context("when starting without the --debugAddr flag", func() {
		BeforeEach(func() {
			client = startGarden()
		})

		It("does not expose the pprof debug endpoint", func() {
			_, err := http.Get(fmt.Sprintf("http://%s/debug/pprof/?debug=1", debugAddr))
			Expect(err).To(HaveOccurred())
		})

		It("does not expose the log level adjustment endpoint", func() {
			_, err := http.Get(fmt.Sprintf("http://%s/log-level -X PUT -d debug", debugAddr))
			Expect(err).To(HaveOccurred())
		})
	})

	Context("when starting with the --debugAddr flag", func() {
		BeforeEach(func() {
			client = startGarden("--debugAddr", debugAddr)
		})

		It("exposes the pprof debug endpoint", func() {
			_, err := http.Get(fmt.Sprintf("http://%s/debug/pprof/?debug=1", debugAddr))
			Expect(err).ToNot(HaveOccurred())
		})

		It("exposes the log level adjustment endpoint", func() {
			_, err := http.Get(fmt.Sprintf("http://%s/log-level -X PUT -d debug", debugAddr))
			Expect(err).ToNot(HaveOccurred())

			_, err = http.Get(fmt.Sprintf("http://%s/log-level -X PUT -d info", debugAddr))
			Expect(err).ToNot(HaveOccurred())

			_, err = http.Get(fmt.Sprintf("http://%s/log-level -X PUT -d error", debugAddr))
			Expect(err).ToNot(HaveOccurred())

			_, err = http.Get(fmt.Sprintf("http://%s/log-level -X PUT -d fatal", debugAddr))
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
