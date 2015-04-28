package domain_bbs_test

import (
	"time"

	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/models"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("DomainBBS", func() {
	Describe("UpsertDomain", func() {
		It("inserts a domain into the bbs", func() {
			err := bbs.UpsertDomain("the-domain", 10)
			Expect(err).NotTo(HaveOccurred())

			_, err = etcdClient.Get(shared.DomainSchemaPath("the-domain"))
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when the domain exists in the bbs", func() {
			BeforeEach(func() {
				err := bbs.UpsertDomain("the-domain", 10)
				Expect(err).NotTo(HaveOccurred())
			})

			It("updates the domain in the bbs", func() {
				nodeBefore, err := etcdClient.Get(shared.DomainSchemaPath("the-domain"))
				Expect(err).NotTo(HaveOccurred())

				err = bbs.UpsertDomain("the-domain", 10)
				Expect(err).NotTo(HaveOccurred())

				nodeAfter, err := etcdClient.Get(shared.DomainSchemaPath("the-domain"))
				Expect(err).NotTo(HaveOccurred())

				Expect(nodeAfter.Index).To(BeNumerically(">", nodeBefore.Index))
			})
		})

		Context("when the domain is invalid", func() {
			It("returns an error", func() {
				err := bbs.UpsertDomain("", 0)
				Expect(err).To(HaveOccurred())
				Expect(err).To(ConsistOf(models.ErrInvalidParameter{"domain"}))
			})
		})

		Context("when the ttlInSeconds is invalid", func() {
			It("returns an error", func() {
				err := bbs.UpsertDomain("domain", -1)
				Expect(err).To(HaveOccurred())
				Expect(err).To(ConsistOf(models.ErrInvalidParameter{"ttlInSeconds"}))
			})
		})
	})

	Describe("Domains", func() {
		Context("when domains exist in the bbs", func() {
			BeforeEach(func() {
				err := bbs.UpsertDomain("the-domain", 2)
				Expect(err).NotTo(HaveOccurred())
				err = bbs.UpsertDomain("another-domain", 10)
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns all domains which have not expired", func() {
				domains, err := bbs.Domains()
				Expect(err).NotTo(HaveOccurred())
				Expect(domains).To(ConsistOf("the-domain", "another-domain"))

				time.Sleep(3001 * time.Millisecond)

				domains, err = bbs.Domains()
				Expect(err).NotTo(HaveOccurred())
				Expect(domains).To(ConsistOf("another-domain"))
			})
		})

		Context("when no domains exist in the bbs", func() {
			It("returns an empty list", func() {
				domains, err := bbs.Domains()
				Expect(err).NotTo(HaveOccurred())
				Expect(domains).To(BeEmpty())
			})
		})
	})
})
