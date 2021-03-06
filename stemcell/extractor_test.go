package stemcell_test

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-init/stemcell"

	fakesys "github.com/cloudfoundry/bosh-agent/system/fakes"

	biproperty "github.com/cloudfoundry/bosh-init/common/property"

	fakebistemcell "github.com/cloudfoundry/bosh-init/stemcell/fakes"
)

var _ = Describe("Manager", func() {
	var (
		extractor           Extractor
		fs                  *fakesys.FakeFileSystem
		reader              *fakebistemcell.FakeStemcellReader
		stemcellTarballPath string
		tempExtractionDir   string

		expectedExtractedStemcell ExtractedStemcell
	)

	BeforeEach(func() {
		fs = fakesys.NewFakeFileSystem()
		reader = fakebistemcell.NewFakeReader()
		stemcellTarballPath = "/stemcell/tarball/path"
		tempExtractionDir = "/path/to/dest"
		fs.TempDirDir = tempExtractionDir

		extractor = NewExtractor(reader, fs)

		expectedExtractedStemcell = NewExtractedStemcell(
			Manifest{
				Name:      "fake-stemcell-name",
				ImagePath: "fake-image-path",
				CloudProperties: biproperty.Map{
					"fake-prop-key": "fake-prop-value",
				},
			},
			tempExtractionDir,
			fs,
		)
		reader.SetReadBehavior(stemcellTarballPath, tempExtractionDir, expectedExtractedStemcell, nil)
	})

	Describe("Extract", func() {
		It("extracts and parses the stemcell manifest", func() {
			stemcell, err := extractor.Extract(stemcellTarballPath)
			Expect(err).ToNot(HaveOccurred())
			Expect(stemcell).To(Equal(expectedExtractedStemcell))

			Expect(reader.ReadInputs).To(Equal([]fakebistemcell.ReadInput{
				{
					StemcellTarballPath: stemcellTarballPath,
					DestPath:            tempExtractionDir,
				},
			}))
		})

		It("when the read fails, returns an error", func() {
			reader.SetReadBehavior(stemcellTarballPath, tempExtractionDir, expectedExtractedStemcell, errors.New("fake-read-error"))

			_, err := extractor.Extract(stemcellTarballPath)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-read-error"))
		})
	})
})
