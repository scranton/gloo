package e2e_test

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	gatewayv1 "github.com/solo-io/gloo/projects/gateway/pkg/api/v1"
	"github.com/solo-io/gloo/test/services"
	"github.com/solo-io/solo-kit/pkg/api/v1/clients"
)

var _ = Describe("Happypath", func() {

	var (
		ctx            context.Context
		cancel         context.CancelFunc
		testClients    services.TestClients
		writeNamespace string
	)

	Describe("in memory", func() {

		BeforeEach(func() {
			ctx, cancel = context.WithCancel(context.Background())

			writeNamespace = "gloo-system"
			ro := &services.RunOptions{
				NsToWrite: writeNamespace,
				NsToWatch: []string{"default", writeNamespace},
				WhatToRun: services.What{
					DisableFds: true,
					DisableUds: true,
				},
			}

			testClients = services.RunGlooGatewayUdsFds(ctx, ro)
		})

		AfterEach(func() {
			cancel()
		})

		It("should create 2 gateway", func() {

			gatewaycli := testClients.GatewayClient

			Eventually(func() (gatewayv1.GatewayList, error) { return gatewaycli.List(writeNamespace, clients.ListOpts{}) }, "5s", "0.1s").Should(HaveLen(2))
			gw, err := gatewaycli.List(writeNamespace, clients.ListOpts{})
			Expect(err).NotTo(HaveOccurred())

			numssl := 0
			if gw[0].Ssl {
				numssl += 1
			}
			if gw[1].Ssl {
				numssl += 1
			}
			Expect(numssl).To(Equal(1))
		})

	})
})
