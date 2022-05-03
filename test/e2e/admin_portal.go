package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"os"

	. "github.com/onsi/ginkgo" //nolint
	. "github.com/onsi/gomega" //nolint
	conditions "github.com/serge1peshcoff/selenium-go-conditions"
	"github.com/tebeka/selenium"
)

var _ = FDescribe("Admin Portal E2E Testing", func() {
	BeforeEach(skipIfNotInDevelopmentEnv)
	var wdPoint *selenium.WebDriver
	var wd selenium.WebDriver
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
		wd.Wait(conditions.ElementIsLocated(selenium.ByCSSSelector, "div[data-automation-key='name']"))

		cluster, err := wd.FindElement(selenium.ByCSSSelector, "div[data-automation-key='name']")
		if err != nil {
			panic(err)
		}

		Expect(cluster.Text()).To(Equal(os.Getenv("CLUSTER")))
	})

	It("Should be able to filter cluster data correctly", func() {
		wd.Wait(conditions.ElementIsLocated(selenium.ByCSSSelector, "div[data-automation-key='name']"))

		filter, err := wd.FindElement(selenium.ByCSSSelector, "input[placeholder='Filter on resource ID']")
		if err != nil {
			panic(err)
		}

		// Set filter so it doesn't match cluster name
		filter.SendKeys("Incorrect Cluster")

		wd.Wait(conditions.ElementIsLocated(selenium.ByCSSSelector, "span.itemsCount-162"))
		text, err := wd.FindElement(selenium.ByCSSSelector, "span.itemsCount-162")
		if err != nil {
			panic(err)
		}

		Expect(text.Text()).To(Equal("Showing 0 items"))
	})

	It("Should be able to populate cluster info panel correctly", func() {
		testValues := [17]string{
			"Public",
			"Undefined",
			"1",
			"Undefined",
			"2021-11-03T06:04:39Z",
			"unknown",
			"Undefined",
			"elljohns-test-hrqbs",
			"Undefined",
			"Undefined",
			"Undefined",
			"Undefined",
			"Undefined",
			"elljohns-test",
			"Succeeded",
			"4.8.11",
			"Installed"}

		wd.Wait(conditions.ElementIsLocated(selenium.ByCSSSelector, "div[data-automation-key='name']"))

		cluster, err := wd.FindElement(selenium.ByCSSSelector, "div[data-automation-key='name']")
		if err != nil {
			panic(err)
		}

		cluster.Click()

		wd.Wait(conditions.ElementIsLocated(selenium.ByCSSSelector, "ms-Panel is-open ms-Panel--hasCloseButton ms-Panel--custom root-225"))

		panelFields, err := wd.FindElements(selenium.ByCSSSelector, "css-287")
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

		panelValues, err := wd.FindElements(selenium.ByCSSSelector, "css-290")
		if err != nil {
			panic(err)
		}

		for i, panelValue := range panelValues {
			panelFieldText := filteredPanelFields[i]

			panelValueText, err := panelValue.Text()
			if err != nil {
				panic(err)
			}

			Expect(panelFieldText + " : " + panelValueText).To(Equal(panelFieldText + " : " + testValues[i]))
		}
	})

	It("Should be able to copy cluster resource id", func() {
		wd.Wait(conditions.ElementIsLocated(selenium.ByCSSSelector, "button[aria-label='Copy Resource ID']"))

		button, err := wd.FindElement(selenium.ByCSSSelector, "button[aria-label='Copy Resource ID']")
		if err != nil {
			panic(err)
		}

		button.Click()

		filter, err := wd.FindElement(selenium.ByCSSSelector, "input[placeholder='Filter on resource ID']")
		if err != nil {
			panic(err)
		}

		// Paste clipboard
		filter.Click()
		filter.SendKeys(selenium.ControlKey + "v")

		resourceId, err := filter.GetAttribute("value")

		if err != nil {
			panic(err)
		}

		Expect(resourceId).To(Equal("/subscriptions/225e02bc-43d0-43d1-a01a-17e584a4ef69/resourceGroups/v4-eastus/providers/Microsoft.RedHatOpenShift/openShiftClusters/" + os.Getenv("CLUSTER")))
	})

	It("Should be able to open ssh panel and get ssh details", func() {
		wd.Wait(conditions.ElementIsLocated(selenium.ByCSSSelector, "button[aria-label='SSH']"))

		button, err := wd.FindElement(selenium.ByCSSSelector, "button[aria-label='SSH']")
		if err != nil {
			panic(err)
		}

		button.Click()

		wd.Wait(conditions.ElementIsLocated(selenium.ByID, "ModalFocusTrapZone25"))

		wd.Wait(conditions.ElementIsLocated(selenium.ByID, "Dropdown55"))
		sshDropdown, err := wd.FindElement(selenium.ByID, "Dropdown55")
		if err != nil {
			panic(err)
		}

		sshDropdown.Click()

		wd.Wait(conditions.ElementIsLocated(selenium.ByID, "Dropdown55-list0"))
		machine, err := wd.FindElement(selenium.ByID, "Dropdown55-list0")
		if err != nil {
			panic(err)
		}

		machine.Click()

		wd.Wait(conditions.ElementIsLocated(selenium.ByID, "id__56"))
		requestBtn, err := wd.FindElement(selenium.ByID, "id__56")
		if err != nil {
			panic(err)
		}

		requestBtn.Click()

		wd.Wait(conditions.ElementIsLocated(selenium.ByID, "title24"))

		// Test fails if these labels aren't present
		err = wd.Wait(conditions.ElementIsLocated(selenium.ByID, "TextFieldLabel72"))
		if err != nil {
			panic(err)
		}

		err = wd.Wait(conditions.ElementIsLocated(selenium.ByID, "TextFieldLabel80"))
		if err != nil {
			panic(err)
		}
	})
})
