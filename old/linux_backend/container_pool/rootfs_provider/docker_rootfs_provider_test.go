package rootfs_provider_test

import (
	"errors"

	"github.com/cloudfoundry-incubator/garden-linux/old/linux_backend/container_pool/fake_graph_driver"
	"github.com/cloudfoundry-incubator/garden-linux/old/linux_backend/container_pool/repository_fetcher"
	"github.com/cloudfoundry-incubator/garden-linux/old/linux_backend/container_pool/repository_fetcher/fake_repository_fetcher"
	. "github.com/cloudfoundry-incubator/garden-linux/old/linux_backend/container_pool/rootfs_provider"
	"github.com/cloudfoundry-incubator/garden-linux/process"
	"github.com/pivotal-golang/lager/lagertest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type FakeVolumeCreator struct {
	Created     []RootAndVolume
	CreateError error
}

type RootAndVolume struct {
	RootPath string
	Volume   string
}

func (f *FakeVolumeCreator) Create(path, v string) error {
	f.Created = append(f.Created, RootAndVolume{path, v})
	return f.CreateError
}

var _ = Describe("DockerRootFSProvider", func() {
	var (
		fakeRepositoryFetcher *fake_repository_fetcher.FakeRepositoryFetcher
		fakeGraphDriver       *fake_graph_driver.FakeGraphDriver
		fakeVolumeCreator     *FakeVolumeCreator
		newRepoFetcher        func(string) (repository_fetcher.RepositoryFetcher, error)

		provider RootFSProvider

		logger *lagertest.TestLogger
	)

	BeforeEach(func() {
		fakeRepositoryFetcher = fake_repository_fetcher.New()
		fakeGraphDriver = &fake_graph_driver.FakeGraphDriver{}
		fakeVolumeCreator = &FakeVolumeCreator{}
		newRepoFetcher = func(_ string) (repository_fetcher.RepositoryFetcher, error) {
			return fakeRepositoryFetcher, nil
		}
		var err error
		provider, err = NewDocker(newRepoFetcher, "dummy", fakeGraphDriver, fakeVolumeCreator)
		Ω(err).ShouldNot(HaveOccurred())

		logger = lagertest.NewTestLogger("test")
	})

	Describe("ProvideRootFS", func() {
		It("fetches it and creates a graph entry with it as the parent", func() {
			fakeRepositoryFetcher.FetchResult = "some-image-id"
			fakeGraphDriver.GetReturns("/some/graph/driver/mount/point", nil)

			mountpoint, envvars, err := provider.ProvideRootFS(logger, "some-id", parseURL("docker:///some-repository-name"))
			Ω(err).ShouldNot(HaveOccurred())

			Ω(fakeGraphDriver.CreateCallCount()).Should(Equal(1))
			id, parent := fakeGraphDriver.CreateArgsForCall(0)
			Ω(id).Should(Equal("some-id"))
			Ω(parent).Should(Equal("some-image-id"))

			Ω(fakeRepositoryFetcher.Fetched()).Should(ContainElement(
				fake_repository_fetcher.FetchSpec{
					Repository: "some-repository-name",
					Tag:        "latest",
				},
			))

			Ω(mountpoint).Should(Equal("/some/graph/driver/mount/point"))
			Ω(envvars).Should(Equal(process.Env{"env1": "env1Value", "env2": "env2Value"}))
		})

		Context("when the image has associated VOLUMEs", func() {
			It("creates empty directories for all volumes", func() {
				fakeRepositoryFetcher.FetchResult = "some-image-id"
				fakeGraphDriver.GetReturns("/some/graph/driver/mount/point", nil)

				_, _, err := provider.ProvideRootFS(logger, "some-id", parseURL("docker:///some-repository-name"))
				Ω(err).ShouldNot(HaveOccurred())

				Ω(fakeVolumeCreator.Created).Should(Equal([]RootAndVolume{{"/some/graph/driver/mount/point", "/foo"}, {"/some/graph/driver/mount/point", "/bar"}}))
			})

			Context("when creating a volume fails", func() {
				It("returns an error", func() {
					fakeRepositoryFetcher.FetchResult = "some-image-id"
					fakeGraphDriver.GetReturns("/some/graph/driver/mount/point", nil)
					fakeVolumeCreator.CreateError = errors.New("o nooo")

					_, _, err := provider.ProvideRootFS(logger, "some-id", parseURL("docker:///some-repository-name"))
					Ω(err).Should(HaveOccurred())
				})
			})
		})

		Context("when the url is missing a path", func() {
			It("returns an error", func() {
				_, _, err := provider.ProvideRootFS(logger, "some-id", parseURL("docker://"))
				Ω(err).Should(Equal(ErrInvalidDockerURL))
			})
		})

		Context("and a tag is specified via a fragment", func() {
			It("uses it when fetching the repository", func() {
				_, _, err := provider.ProvideRootFS(logger, "some-id", parseURL("docker:///some-repository-name#some-tag"))
				Ω(err).ShouldNot(HaveOccurred())

				Ω(fakeRepositoryFetcher.Fetched()).Should(ContainElement(
					fake_repository_fetcher.FetchSpec{
						Repository: "some-repository-name",
						Tag:        "some-tag",
					},
				))
			})
		})

		Context("and a host is specified", func() {
			var registryName string

			BeforeEach(func() {
				fakeRepositoryFetcher = fake_repository_fetcher.New()
				newRepoFetcher = func(regName string) (repository_fetcher.RepositoryFetcher, error) {
					registryName = regName
					return fakeRepositoryFetcher, nil
				}
				var err error
				provider, err = NewDocker(newRepoFetcher, "default.registry", fakeGraphDriver, fakeVolumeCreator)
				Ω(err).ShouldNot(HaveOccurred())
			})

			It("uses the host as the registry when fetching the repository", func() {
				_, _, err := provider.ProvideRootFS(logger, "some-id", parseURL("docker://some.host/some-repository-name"))
				Ω(err).ShouldNot(HaveOccurred())
				Ω(registryName).Should(Equal("some.host"))
			})

			Context("and the repository fetcher could not be created", func() {
				It("for the default registry", func() {
					newRepoFetcher = func(regName string) (repository_fetcher.RepositoryFetcher, error) {
						return nil, errors.New("failed")
					}

					_, err := NewDocker(newRepoFetcher, "default.registry", fakeGraphDriver, fakeVolumeCreator)
					Ω(err).Should(MatchError("failed"))
				})

				It("for a specified registry", func() {
					newRepoFetcher = func(regName string) (repository_fetcher.RepositoryFetcher, error) {
						if regName == "some.host" {
							return nil, errors.New("failed")
						}
						return fakeRepositoryFetcher, nil
					}
					provider, err := NewDocker(newRepoFetcher, "default.registry", fakeGraphDriver, fakeVolumeCreator)
					Ω(err).ShouldNot(HaveOccurred())

					_, _, err = provider.ProvideRootFS(logger, "some-id", parseURL("docker://some.host/some-repository-name"))
					Ω(err).Should(MatchError(ErrInvalidDockerURL))
				})

			})
		})

		Context("but fetching it fails", func() {
			disaster := errors.New("oh no!")

			BeforeEach(func() {
				fakeRepositoryFetcher.FetchError = disaster
			})

			It("returns the error", func() {
				_, _, err := provider.ProvideRootFS(logger, "some-id", parseURL("docker:///some-repository-name"))
				Ω(err).Should(Equal(disaster))
			})
		})

		Context("but creating the graph entry fails", func() {
			disaster := errors.New("oh no!")

			BeforeEach(func() {
				fakeGraphDriver.CreateReturns(disaster)
			})

			It("returns the error", func() {
				_, _, err := provider.ProvideRootFS(logger, "some-id", parseURL("docker:///some-repository-name#some-tag"))
				Ω(err).Should(Equal(disaster))
			})
		})

		Context("but getting the graph entry fails", func() {
			disaster := errors.New("oh no!")

			BeforeEach(func() {
				fakeGraphDriver.GetReturns("", disaster)
			})

			It("returns the error", func() {
				_, _, err := provider.ProvideRootFS(logger, "some-id", parseURL("docker:///some-repository-name#some-tag"))
				Ω(err).Should(Equal(disaster))
			})
		})
	})

	Describe("CleanupRootFS", func() {
		It("removes the container from the rootfs graph", func() {
			err := provider.CleanupRootFS(logger, "some-id")
			Ω(err).ShouldNot(HaveOccurred())

			Ω(fakeGraphDriver.PutCallCount()).Should(Equal(1))
			putted := fakeGraphDriver.PutArgsForCall(0)
			Ω(putted).Should(Equal("some-id"))

			Ω(fakeGraphDriver.RemoveCallCount()).Should(Equal(1))
			removed := fakeGraphDriver.RemoveArgsForCall(0)
			Ω(removed).Should(Equal("some-id"))
		})

		Context("when removing the container from the graph fails", func() {
			disaster := errors.New("oh no!")

			BeforeEach(func() {
				fakeGraphDriver.RemoveReturns(disaster)
			})

			It("returns the error", func() {
				err := provider.CleanupRootFS(logger, "some-id")
				Ω(err).Should(Equal(disaster))
			})
		})
	})
})
