package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"os"
	"time"

	. "github.com/onsi/ginkgo"                           //nolint
	. "github.com/onsi/gomega"                           //nolint
	. "github.com/serge1peshcoff/selenium-go-conditions" //nolint
	. "github.com/tebeka/selenium"                       //nolint
)

var _ = Describe("Admin Portal E2E Testing", func() {
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
		err := wd.Wait(ElementIsLocated(ByCSSSelector, "div[data-automation-key='name']"))
		if err != nil {
			TakeScreenshot(wd, err)
		}

		cluster, err := wd.FindElement(ByCSSSelector, "div[data-automation-key='name']")
		if err != nil {
			TakeScreenshot(wd, err)
		}

		Expect(cluster.Text()).To(Equal(os.Getenv("CLUSTER")))
	})

	It("Should be able to filter cluster data correctly", func() {
		wd.Wait(ElementIsLocated(ByCSSSelector, "div[data-automation-key='name']"))

		filter, err := wd.FindElement(ByCSSSelector, "input[placeholder='Filter on resource ID']")
		if err != nil {
			TakeScreenshot(wd, err)
		}

		// Set filter so it doesn't match cluster name
		filter.SendKeys("Incorrect Cluster")

		wd.Wait(ElementIsLocated(ByID, "ClusterCount"))
		text, err := wd.FindElement(ByID, "ClusterCount")
		if err != nil {
			TakeScreenshot(wd, err)
		}

		Expect(text.Text()).To(Equal("Showing 0 items"))
	})

	It("Should be able to populate cluster info panel correctly", func() {
		const CLUSTER_INFO_HEADINGS = 17

		err := wd.Wait(ElementIsLocated(ByCSSSelector, "div[data-automation-key='name']"))
		if err != nil {
			TakeScreenshot(wd, err)
		}

		cluster, err := wd.FindElement(ByCSSSelector, "div[data-automation-key='name']")
		if err != nil {
			TakeScreenshot(wd, err)
		}

		err = cluster.Click()
		if err != nil {
			TakeScreenshot(wd, err)
		}

		err = wd.WaitWithTimeout(ElementIsLocated(ByID, "ClusterDetailCell"), 2*time.Minute)
		if err != nil {
			TakeScreenshot(wd, err)
		}

		panelSpans, err := wd.FindElements(ByID, "ClusterDetailCell")
		if err != nil {
			TakeScreenshot(wd, err)
		}

		Expect(len(panelSpans)).To(Equal(CLUSTER_INFO_HEADINGS * 3))

		panelFields := panelSpans[0 : CLUSTER_INFO_HEADINGS-1]
		panelColons := panelSpans[CLUSTER_INFO_HEADINGS : CLUSTER_INFO_HEADINGS*2-1]
		panelValues := panelSpans[CLUSTER_INFO_HEADINGS*2 : len(panelSpans)-1]

		for _, panelField := range panelFields {
			panelText, err := panelField.Text()
			if err != nil {
				TakeScreenshot(wd, err)
			}

			Expect(panelText).To(Not(Equal("")))
		}

		for _, panelField := range panelColons {
			panelText, err := panelField.Text()
			if err != nil {
				TakeScreenshot(wd, err)
			}

			Expect(panelText).To(Equal(":"))
		}

		for _, panelField := range panelValues {
			panelText, err := panelField.Text()
			if err != nil {
				TakeScreenshot(wd, err)
			}

			Expect(panelText).To(Not(Equal("")))
		}
	})

	It("Should be able to copy cluster resource id", func() {
		wd.Wait(ElementIsLocated(ByCSSSelector, "button[aria-label='Copy Resource ID']"))

		button, err := wd.FindElement(ByCSSSelector, "button[aria-label='Copy Resource ID']")
		if err != nil {
			TakeScreenshot(wd, err)
		}

		button.Click()

		filter, err := wd.FindElement(ByCSSSelector, "input[placeholder='Filter on resource ID']")
		if err != nil {
			TakeScreenshot(wd, err)
		}

		// Paste clipboard
		filter.Click()
		filter.SendKeys(ControlKey + "v")
		resourceId, err := filter.GetAttribute("value")

		if err != nil {
			TakeScreenshot(wd, err)
		}

		Expect(resourceId).To(ContainSubstring("/providers/Microsoft.RedHatOpenShift/openShiftClusters/" + os.Getenv("CLUSTER")))
	})

	It("Should be able to open ssh panel and get ssh details", func() {
		wd.Wait(ElementIsLocated(ByCSSSelector, "button[aria-label='SSH']"))

		button, err := wd.FindElement(ByCSSSelector, "button[aria-label='SSH']")
		if err != nil {
			TakeScreenshot(wd, err)
		}

		button.Click()

		wd.Wait(ElementIsLocated(ByID, "sshModal"))
		wd.Wait(ElementIsLocated(ByID, "sshDropdown"))

		sshDropdown, err := wd.FindElement(ByID, "sshDropdown")
		if err != nil {
			TakeScreenshot(wd, err)
		}

		sshDropdown.Click()

		wd.Wait(ElementIsLocated(ByID, "sshDropdown-list0"))
		machine, err := wd.FindElement(ByID, "sshDropdown-list0")
		if err != nil {
			TakeScreenshot(wd, err)
		}

		machine.Click()

		wd.Wait(ElementIsLocated(ByID, "sshButton"))
		requestBtn, err := wd.FindElement(ByID, "sshButton")
		if err != nil {
			TakeScreenshot(wd, err)
		}

		requestBtn.Click()

		// Test fails if these labels aren't present
		err = wd.Wait(ElementIsLocated(ByID, "sshCommand"))
		if err != nil {
			TakeScreenshot(wd, err)
		}
	})
})
