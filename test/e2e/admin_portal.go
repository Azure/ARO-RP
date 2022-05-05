package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"os"

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
		wd.Wait(ElementIsLocated(ByCSSSelector, "div[data-automation-key='name']"))

		cluster, err := wd.FindElement(ByCSSSelector, "div[data-automation-key='name']")
		if err != nil {
			panic(err)
		}

		Expect(cluster.Text()).To(Equal(os.Getenv("CLUSTER")))
	})

	It("Should be able to filter cluster data correctly", func() {
		wd.Wait(ElementIsLocated(ByCSSSelector, "div[data-automation-key='name']"))

		filter, err := wd.FindElement(ByCSSSelector, "input[placeholder='Filter on resource ID']")
		if err != nil {
			panic(err)
		}

		// Set filter so it doesn't match cluster name
		filter.SendKeys("Incorrect Cluster")

		wd.Wait(ElementIsLocated(ByCSSSelector, "span.itemsCount-162"))
		text, err := wd.FindElement(ByCSSSelector, "span.itemsCount-162")
		if err != nil {
			panic(err)
		}

		Expect(text.Text()).To(Equal("Showing 0 items"))
	})

	FIt("Should be able to populate cluster info panel correctly", func() {
		err := wd.Wait(ElementIsLocated(ByCSSSelector, "div[data-automation-key='name']"))
		if err != nil {
			panic(err)
		}

		cluster, err := wd.FindElement(ByCSSSelector, "div[data-automation-key='name']")
		if err != nil {
			panic(err)
		}

		err = cluster.Click()
		if err != nil {
			panic(err)
		}

		err = wd.Wait(ElementIsLocated(ByCSSSelector, "div.css-113"))
		if err != nil {
			panic(err)
		}

		err = wd.Wait(ElementIsLocated(ByCSSSelector, "span.css-287"))
		if err != nil {
			panic(err)
		}

		panelFields, err := wd.FindElements(ByCSSSelector, "span.css-287")
		if err != nil {
			panic(err)
		}
		var filteredPanelFields []string
		for _, panelField := range panelFields {
			panelText, err := panelField.Text()
			if err != nil {
				panic(err)
			}

			if panelText != ":" {
				filteredPanelFields = append(filteredPanelFields, panelText)
			}
		}

		Expect(panelFields).ShouldNot(BeEmpty())

		panelValues, err := wd.FindElements(ByCSSSelector, "span.css-290")
		if err != nil {
			panic(err)
		}

		Expect(len(panelValues)).To(Equal(len(filteredPanelFields)))

		for _, panelValue := range panelValues {
			panelValueText, err := panelValue.Text()
			if err != nil {
				panic(err)
			}

			Expect(panelValueText).To(Not(BeNil()))
		}
	})

	It("Should be able to copy cluster resource id", func() {
		wd.Wait(ElementIsLocated(ByCSSSelector, "button[aria-label='Copy Resource ID']"))

		button, err := wd.FindElement(ByCSSSelector, "button[aria-label='Copy Resource ID']")
		if err != nil {
			panic(err)
		}

		button.Click()

		filter, err := wd.FindElement(ByCSSSelector, "input[placeholder='Filter on resource ID']")
		if err != nil {
			panic(err)
		}

		// Paste clipboard
		filter.Click()
		filter.SendKeys(ControlKey + "v")

		resourceId, err := filter.GetAttribute("value")

		if err != nil {
			panic(err)
		}

		Expect(resourceId).To(ContainSubstring("/providers/Microsoft.RedHatOpenShift/openShiftClusters/" + os.Getenv("CLUSTER")))
	})

	It("Should be able to open ssh panel and get ssh details", func() {
		wd.Wait(ElementIsLocated(ByCSSSelector, "button[aria-label='SSH']"))

		button, err := wd.FindElement(ByCSSSelector, "button[aria-label='SSH']")
		if err != nil {
			panic(err)
		}

		button.Click()

		wd.Wait(ElementIsLocated(ByID, "ModalFocusTrapZone25"))

		wd.Wait(ElementIsLocated(ByID, "Dropdown55"))
		sshDropdown, err := wd.FindElement(ByID, "Dropdown55")
		if err != nil {
			panic(err)
		}

		sshDropdown.Click()

		wd.Wait(ElementIsLocated(ByID, "Dropdown55-list0"))
		machine, err := wd.FindElement(ByID, "Dropdown55-list0")
		if err != nil {
			panic(err)
		}

		machine.Click()

		wd.Wait(ElementIsLocated(ByID, "id__56"))
		requestBtn, err := wd.FindElement(ByID, "id__56")
		if err != nil {
			panic(err)
		}

		requestBtn.Click()

		wd.Wait(ElementIsLocated(ByID, "title24"))

		// Test fails if these labels aren't present
		err = wd.Wait(ElementIsLocated(ByID, "TextFieldLabel72"))
		if err != nil {
			panic(err)
		}

		err = wd.Wait(ElementIsLocated(ByID, "TextFieldLabel80"))
		if err != nil {
			panic(err)
		}
	})
})
