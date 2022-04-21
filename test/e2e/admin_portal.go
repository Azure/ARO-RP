package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	conditions "github.com/serge1peshcoff/selenium-go-conditions"
	. "github.com/tebeka/selenium"
)

var _ = FDescribe("Admin Portal E2E Testing", func() {
	BeforeEach(skipIfNotInDevelopmentEnv)
	var wdPoint *WebDriver
	var wd WebDriver
	var host string
	JustBeforeEach(func() {
		host, wdPoint = adminPortalSessionSetup()
		wd = *wdPoint
		wd.Get(host + "/v2")
		wd.Refresh()
	})

	AfterEach(func() {
		if wd != nil {
			wd.Quit()
		}
	})
	It("Should be able to populate cluster data correctly", func() {
		wd.Wait(conditions.ElementIsLocated(ByCSSSelector, "div[data-automation-key='name']"))

		cluster, err := wd.FindElement(ByCSSSelector, "div[data-automation-key='name']")
		if err != nil {
			panic(err)
		}

		Expect(cluster.Text()).To(Equal("elljohns-test"))
	})

	It("Should be able to filter cluster data correctly", func() {
		wd.Wait(conditions.ElementIsLocated(ByCSSSelector, "div[data-automation-key='name']"))

		filter, err := wd.FindElement(ByCSSSelector, "input[placeholder='Filter on resource ID']")
		if err != nil {
			panic(err)
		}

		filter.SendKeys("boogers")

		wd.Wait(conditions.ElementIsLocated(ByCSSSelector, "span.itemsCount-162"))
		text, err := wd.FindElement(ByCSSSelector, "span.itemsCount-162")
		// wd.Wait(conditions.ElementTextIs()
		// cluster, err := wd.FindElement(ByCSSSelector, "div[data-automation-key='name']")

		Expect(text.Text()).To(Equal("Showing 0 items"))
	})
})
