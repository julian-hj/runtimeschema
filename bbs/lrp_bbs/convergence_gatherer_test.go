package lrp_bbs_test

import (
	"fmt"

	"github.com/cloudfoundry-incubator/runtime-schema/bbs/bbserrors"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/lrp_bbs"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/storeadapter"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type gatherTestData struct {
	actualsToKeep   models.ActualLRPsByProcessGuidAndIndex
	actualsToPrune  models.ActualLRPsByProcessGuidAndIndex
	desiredsToKeep  models.DesiredLRPsByProcessGuid
	desiredsToPrune models.DesiredLRPsByProcessGuid
	domains         models.DomainSet
	cells           models.CellSet
}

func newGatherTestData() *gatherTestData {
	return &gatherTestData{
		actualsToKeep:   models.ActualLRPsByProcessGuidAndIndex{},
		actualsToPrune:  models.ActualLRPsByProcessGuidAndIndex{},
		desiredsToKeep:  models.DesiredLRPsByProcessGuid{},
		desiredsToPrune: models.DesiredLRPsByProcessGuid{},
		domains:         models.DomainSet{},
		cells:           models.CellSet{},
	}
}

var _ = Describe("Convergence", func() {
	Describe("Gathering", func() {
		var gatherTest *gatherTestData
		cellID := "some-cell-id"
		domain := "test-domain"
		processGuid := "process-guid"

		BeforeEach(func() {

			gatherTest = newGatherTestData()

			// DesiredLRPs
			// keep valid desiredLRP
			lrp := newDesiredLRP(processGuid, domain, 4)
			gatherTest.desiredsToKeep.Add(lrp)
			// prune invalid desiredLRP
			gatherTest.desiredsToPrune.Add(models.DesiredLRP{ProcessGuid: "invalid-desired-1"})
			gatherTest.desiredsToPrune.Add(models.DesiredLRP{ProcessGuid: "invalid-desired-2"})

			invalidLRP := newClaimedActualLRP(lrp, cellID, 10)
			invalidLRP.Since = 0 // not valid

			// ActualLRPs
			// keep valid unclaimed for valid desiredlrp
			gatherTest.actualsToKeep.Add(newUnclaimedActualLRP(lrp, 0))
			// keep valid claimed on present cell for valid desiredlrp
			gatherTest.actualsToKeep.Add(newClaimedActualLRP(lrp, cellID, 1))
			// keep valid running on present cell for valid desiredlrp
			gatherTest.actualsToKeep.Add(newRunningActualLRP(lrp, cellID, 2))
			// keep valid running on present cell for missing desiredlrp
			gatherTest.actualsToKeep.Add(newRunningActualLRP(newDesiredLRP("missing-process", domain, 1), cellID, 0))
			// prune invalid lrp
			gatherTest.actualsToPrune.Add(invalidLRP)
			// keep valid crashed for valid desiredlrp
			gatherTest.actualsToKeep.Add(newStartableCrashedActualLRP(lrp, 6))

			// Domains
			gatherTest.domains.Add(domain)

			// Cells
			gatherTest.cells.Add(newCellPresence(cellID))
			gatherTest.cells.Add(newCellPresence("other-cell"))

			createGatherTestData(gatherTest)
		})

		var input *lrp_bbs.ConvergenceInput
		var gatherError error

		JustBeforeEach(func() {
			input, gatherError = bbs.GatherAndPruneLRPConvergenceInput(logger, servicesBBS.NewCellsLoader())
		})

		It("gets all processGuids in the system", func() {
			expectedGuids := map[string]struct{}{
				processGuid:       struct{}{},
				"missing-process": struct{}{},
			}
			Expect(input.AllProcessGuids).To(Equal(expectedGuids))
		})

		It("gets all valid desired LRPs", func() {
			Expect(input.DesiredLRPs).To(HaveLen(len(gatherTest.desiredsToKeep)))

			gatherTest.desiredsToKeep.Each(func(expected models.DesiredLRP) {
				desired, ok := input.DesiredLRPs[expected.ProcessGuid]
				Expect(ok).To(BeTrue(), fmt.Sprintf("expected desiredLRP for process '%s' to be present", expected.ProcessGuid))
				Expect(desired).To(Equal(expected))
			})
		})

		It("prunes the correct desired LRPs", func() {
			gatherTest.desiredsToPrune.Each(func(expected models.DesiredLRP) {
				_, err := bbs.DesiredLRPByProcessGuid(expected.ProcessGuid)
				Expect(err).To(Equal(bbserrors.ErrStoreResourceNotFound))
			})
		})

		It("fetches the correct actualLRPs", func() {
			Expect(input.ActualLRPs).To(HaveLen(len(gatherTest.actualsToKeep)))

			gatherTest.actualsToKeep.Each(func(expected models.ActualLRP) {
				actualIndex, ok := input.ActualLRPs[expected.ProcessGuid]
				Expect(ok).To(BeTrue(), fmt.Sprintf("expected actualIndex for process '%s' to be present", expected.ProcessGuid))
				actual, ok := actualIndex[expected.Index]
				Expect(ok).To(BeTrue(), fmt.Sprintf("expected actual for process '%s' and index %d to be present", expected.ProcessGuid, expected.Index))
				Expect(actual).To(Equal(actual))
			})
		})

		It("prunes the correct actualLRPs", func() {
			gatherTest.actualsToPrune.Each(func(expected models.ActualLRP) {
				_, err := bbs.ActualLRPGroupByProcessGuidAndIndex(expected.ProcessGuid, expected.Index)
				Expect(err).To(Equal(bbserrors.ErrStoreResourceNotFound))
			})
		})

		It("gets all the domains", func() {
			Expect(input.Domains).To(HaveLen(len(gatherTest.domains)))
			gatherTest.domains.Each(func(domain string) {
				Expect(input.Domains).To(HaveKey(domain))
			})
		})

		It("gets all the cells", func() {
			Expect(input.Cells).To(HaveLen(len(gatherTest.cells)))
			gatherTest.cells.Each(func(cell models.CellPresence) {
				Expect(input.Cells).To(ContainElement(cell))
			})
		})
	})
})

func createGatherTestData(test *gatherTestData) {
	test.desiredsToKeep.Each(func(desired models.DesiredLRP) {
		setRawDesiredLRP(desired)
	})

	test.desiredsToPrune.Each(func(desired models.DesiredLRP) {
		createMalformedDesiredLRP(desired.ProcessGuid)
	})

	test.actualsToKeep.Each(func(actual models.ActualLRP) {
		setRawActualLRP(actual)
	})

	test.actualsToPrune.Each(func(actual models.ActualLRP) {
		setRawActualLRP(actual)
	})

	test.domains.Each(func(domain string) {
		createRawDomain(domain)
	})

	test.cells.Each(func(cell models.CellPresence) {
		registerCell(cell)
	})
}
func createMalformedDesiredLRP(guid string) {
	createMalformedValueForKey(shared.DesiredLRPSchemaPath(models.DesiredLRP{ProcessGuid: guid}))
}

func createMalformedActualLRP(guid string, index int) {
	createMalformedValueForKey(shared.ActualLRPSchemaPath(guid, index))
}

func createMalformedValueForKey(key string) {
	err := etcdClient.Create(storeadapter.StoreNode{
		Key:   key,
		Value: []byte("ßßßßßß"),
	})

	Expect(err).NotTo(HaveOccurred())
}

func newCellPresence(cellID string) models.CellPresence {
	return models.NewCellPresence(cellID, "1.2.3.4", "az-1", models.CellCapacity{128, 1024, 3})
}

func newDesiredLRP(guid, domain string, instances int) models.DesiredLRP {
	return models.DesiredLRP{
		Domain:      domain,
		ProcessGuid: guid,
		Instances:   instances,
		RootFS:      "some:rootfs",
		MemoryMB:    1024,
		DiskMB:      512,
		CPUWeight:   42,
		Action:      &models.RunAction{Path: "ls"},
	}
}

func newUnclaimedActualLRP(d models.DesiredLRP, index int) models.ActualLRP {
	return models.ActualLRP{
		ActualLRPKey: models.NewActualLRPKey(d.ProcessGuid, index, d.Domain),
		State:        models.ActualLRPStateUnclaimed,
		Since:        1138,
	}
}

func newClaimedActualLRP(d models.DesiredLRP, cellID string, index int) models.ActualLRP {
	return models.ActualLRP{
		ActualLRPKey:         models.NewActualLRPKey(d.ProcessGuid, index, d.Domain),
		ActualLRPInstanceKey: models.NewActualLRPInstanceKey("instance-guid", cellID),
		State:                models.ActualLRPStateClaimed,
		Since:                1138,
	}
}

func newRunningActualLRP(d models.DesiredLRP, cellID string, index int) models.ActualLRP {
	return models.ActualLRP{
		ActualLRPKey:         models.NewActualLRPKey(d.ProcessGuid, index, d.Domain),
		ActualLRPInstanceKey: models.NewActualLRPInstanceKey("instance-guid", cellID),
		ActualLRPNetInfo:     models.NewActualLRPNetInfo("1.2.3.4", []models.PortMapping{}),
		State:                models.ActualLRPStateRunning,
		Since:                1138,
	}
}

func newStartableCrashedActualLRP(d models.DesiredLRP, index int) models.ActualLRP {
	return models.ActualLRP{
		ActualLRPKey: models.NewActualLRPKey(d.ProcessGuid, index, d.Domain),
		CrashCount:   1,
		State:        models.ActualLRPStateCrashed,
		Since:        1138,
	}
}

func newUnstartableCrashedActualLRP(d models.DesiredLRP, index int) models.ActualLRP {
	return models.ActualLRP{
		ActualLRPKey: models.NewActualLRPKey(d.ProcessGuid, index, d.Domain),
		CrashCount:   201,
		State:        models.ActualLRPStateCrashed,
		Since:        1138,
	}
}
