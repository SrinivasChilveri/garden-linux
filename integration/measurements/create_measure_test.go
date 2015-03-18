package measurements_test

import (
	"runtime"
	"strconv"

	"github.com/cloudfoundry-incubator/garden"
	gclient "github.com/cloudfoundry-incubator/garden/client"
	"github.com/cloudfoundry-incubator/garden/client/connection"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
	creates       = 10 // e.g. 10
	createSamples = 1  // e.g. 5
)

var _ = FDescribe("Container creation", func() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	var (
		create   func(i int, b Benchmarker) string
		goCreate func(i int, b Benchmarker) chan string
	)

	BeforeEach(func() {
		client = gclient.New(connection.New("tcp", "localhost:7777"))

		create = func(i int, b Benchmarker) string {
			var handle string
			b.Time("total create", func() {
				b.Time("create-"+strconv.Itoa(i), func() {
					var ctr garden.Container
					b.Time("total create", func() {
						b.Time("create-"+strconv.Itoa(i), func() {
							var err error
							ctr, err = client.Create(garden.ContainerSpec{})
							Ω(err).ShouldNot(HaveOccurred())
						})
					})
					handle = ctr.Handle()
				})
			})
			return handle
		}

		goCreate = func(i int, b Benchmarker) chan string {
			handleChan := make(chan string, 1)
			go func() {
				defer GinkgoRecover()
				handleChan <- create(i, b)
			}()
			return handleChan
		}
	})

	Measure("multiple concurrent creates", func(b Benchmarker) {
		b.Time("create concurrently "+strconv.Itoa(creates)+" times", func() {
			chans := make([]chan string, creates)

			for i, _ := range chans {
				chans[i] = goCreate(i, b)
			}

			for i, _ := range chans {
				Ω(client.Destroy(<-chans[i])).Should(Succeed())
			}
		})
	}, createSamples)

})
