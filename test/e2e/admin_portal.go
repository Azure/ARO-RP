package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"os"
	"regexp"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	conditions "github.com/serge1peshcoff/selenium-go-conditions"
	"github.com/tebeka/selenium"
	"golang.org/x/crypto/ssh"
)

var _ = Describe("Admin Portal E2E Testing", func() {
	BeforeEach(
		func() {
			skipIfNotInDevelopmentEnv()
			skipIfSeleniumNotEnabled()
		},
	)
	var wdPoint *selenium.WebDriver
	var wd selenium.WebDriver
	var host string

	var clusterDetailTabs = []string{"Overview", "Nodes", "Machines", "MachineSets", "APIStatistics", "KCMStatistics", "DNSStatistics", "IngressStatistics", "ClusterOperators"}

	JustBeforeEach(func() {
		host, wdPoint = adminPortalSessionSetup()
		wd = *wdPoint
		wd.Get(host + "/")
		wd.Refresh()
	})

	JustAfterEach(func() {
		if CurrentSpecReport().Failed() {
			if wd != nil {
				SaveScreenshot(wd, errors.New(CurrentSpecReport().FailureMessage()))
			}
		}
	})

	AfterEach(func() {
		if wd != nil {
			wd.Quit()
		}
	})

	It("Should be able to populate cluster data correctly", func() {
		err := wd.Wait(conditions.ElementIsLocated(selenium.ByCSSSelector, "div[data-automation-key='name']"))
		Expect(err).ToNot(HaveOccurred())

		cluster, err := wd.FindElement(selenium.ByCSSSelector, "div[data-automation-key='name']")
		Expect(err).ToNot(HaveOccurred())

		Expect(cluster.Text()).To(Equal(os.Getenv("CLUSTER")))
	})

	It("Should be able to filter cluster data correctly", func() {
		wd.Wait(conditions.ElementIsLocated(selenium.ByCSSSelector, "div[data-automation-key='name']"))

		filter, err := wd.FindElement(selenium.ByCSSSelector, "input[placeholder='Filter on resource ID']")
		Expect(err).ToNot(HaveOccurred())

		// Set filter so it doesn't match cluster name
		filter.SendKeys("Incorrect Cluster")

		wd.Wait(conditions.ElementIsLocated(selenium.ByID, "ClusterCount"))
		text, err := wd.FindElement(selenium.ByID, "ClusterCount")
		Expect(err).ToNot(HaveOccurred())

		Expect(text.Text()).To(Equal("Showing 0 items"))
	})

	It("Should be able to populate cluster info panel correctly", func() {
		err := wd.Wait(conditions.ElementIsLocated(selenium.ByCSSSelector, "div[data-automation-key='name']"))
		Expect(err).ToNot(HaveOccurred())

		cluster, err := wd.FindElement(selenium.ByCSSSelector, "div[data-automation-key='name']")
		Expect(err).ToNot(HaveOccurred())

		err = cluster.Click()
		Expect(err).ToNot(HaveOccurred())

		err = wd.WaitWithTimeout(conditions.ElementIsLocated(selenium.ByCSSSelector, ".clusterOverviewList"), 2*time.Minute)
		Expect(err).ToNot(HaveOccurred())

		err = wd.WaitWithTimeout(noContentIsLoading, 2*time.Minute)
		Expect(err).ToNot(HaveOccurred())

		list, err := wd.FindElement(selenium.ByCSSSelector, ".clusterOverviewList")
		Expect(err).ToNot(HaveOccurred())

		expectedProperties := []string{
			"ApiServer Visibility",
			"ApiServer URL",
			"Architecture Version",
			"Console Link",
			"Created At",
			"Created By",
			"Failed Provisioning State",
			"Infra Id",
			"Last Admin Update Error",
			"Last Modified At",
			"Last Modified By",
			"Last Provisioning State",
			"Location",
			"Name",
			"Provisioning State",
			"Resource Id",
			"Version",
			"Installation Status",
		}

		Eventually(func(g Gomega) {
			for i, wantName := range expectedProperties {
				cell, err := list.FindElement(selenium.ByCSSSelector, fmt.Sprintf("div[data-automationid='ListCell'][data-list-index='%d']", i))
				Expect(err).ToNot(HaveOccurred())

				name, err := cell.FindElement(selenium.ByCSSSelector, "div[data-automationid='DetailsRowCell'][data-automation-key='name']")
				Expect(err).ToNot(HaveOccurred())
				Expect(name.Text()).To(Equal(wantName))

				value, err := cell.FindElement(selenium.ByCSSSelector, "div[data-automationid='DetailsRowCell'][data-automation-key='value']")
				Expect(err).ToNot(HaveOccurred())
				Expect(value.Text()).To(Not(Equal("")))
			}
		}).WithTimeout(time.Minute).WithPolling(time.Second).Should(Succeed())
	})

	It("Should be able to copy cluster resource id", func() {
		wd.Wait(conditions.ElementIsLocated(selenium.ByCSSSelector, "button[aria-label='Copy Resource ID']"))

		button, err := wd.FindElement(selenium.ByCSSSelector, "button[aria-label='Copy Resource ID']")
		Expect(err).ToNot(HaveOccurred())

		button.Click()

		filter, err := wd.FindElement(selenium.ByCSSSelector, "input[placeholder='Filter on resource ID']")
		Expect(err).ToNot(HaveOccurred())

		// Paste clipboard
		filter.Click()
		filter.SendKeys(selenium.ControlKey + "v")
		resourceId, err := filter.GetAttribute("value")
		Expect(err).ToNot(HaveOccurred())

		Expect(resourceId).To(ContainSubstring("/providers/Microsoft.RedHatOpenShift/openShiftClusters/" + os.Getenv("CLUSTER")))
	})

	It("Should be able to open ssh panel and get ssh details", func() {
		wd.Wait(conditions.ElementIsLocated(selenium.ByCSSSelector, "button[aria-label='SSH']"))

		button, err := wd.FindElement(selenium.ByCSSSelector, "button[aria-label='SSH']")
		Expect(err).ToNot(HaveOccurred())

		button.Click()

		wd.Wait(conditions.ElementIsLocated(selenium.ByID, "sshModal"))
		wd.Wait(conditions.ElementIsLocated(selenium.ByID, "sshDropdown"))

		sshDropdown, err := wd.FindElement(selenium.ByID, "sshDropdown")
		Expect(err).ToNot(HaveOccurred())

		sshDropdown.Click()

		wd.Wait(conditions.ElementIsLocated(selenium.ByID, "sshDropdown-list0"))
		machine, err := wd.FindElement(selenium.ByID, "sshDropdown-list0")
		Expect(err).ToNot(HaveOccurred())

		machine.Click()

		wd.Wait(conditions.ElementIsLocated(selenium.ByID, "sshButton"))
		requestBtn, err := wd.FindElement(selenium.ByID, "sshButton")
		Expect(err).ToNot(HaveOccurred())

		requestBtn.Click()

		// Test fails if these labels aren't present
		err = wd.Wait(conditions.ElementIsLocated(selenium.ByID, "sshCommand"))
		Expect(err).ToNot(HaveOccurred())
	})

	It("Should be able to open ssh panel and get ssh details, then SSH to the cluster", Pending, func() {
		// TODO: See if we can make this work. Currently it fails with a blank SSH command
		// TODO: This may be due to insufficient permissions. Further work needed.
		wd.Wait(conditions.ElementIsLocated(selenium.ByCSSSelector, "button[aria-label='SSH']"))

		button, err := wd.FindElement(selenium.ByCSSSelector, "button[aria-label='SSH']")
		Expect(err).ToNot(HaveOccurred(), "SSH button should have been found")

		button.Click()

		wd.Wait(conditions.ElementIsLocated(selenium.ByID, "sshModal"))
		wd.Wait(conditions.ElementIsLocated(selenium.ByID, "sshDropdown"))

		sshDropdown, err := wd.FindElement(selenium.ByID, "sshDropdown")
		Expect(err).ToNot(HaveOccurred(), "SSH dropdown should have been found")

		sshDropdown.Click()

		wd.Wait(conditions.ElementIsLocated(selenium.ByID, "sshDropdown-list0"))
		machine, err := wd.FindElement(selenium.ByID, "sshDropdown-list0")
		Expect(err).ToNot(HaveOccurred(), "SSH machine should have been found")

		machine.Click()

		wd.Wait(conditions.ElementIsLocated(selenium.ByID, "sshButton"))
		requestBtn, err := wd.FindElement(selenium.ByID, "sshButton")
		Expect(err).ToNot(HaveOccurred(), "SSH request button should have been found")

		requestBtn.Click()

		// Test fails if these labels aren't present
		wd.Wait(conditions.ElementIsLocated(selenium.ByID, "sshCommand"))
		sshCommand, err := wd.FindElement(selenium.ByID, "sshCommand")
		Expect(err).ToNot(HaveOccurred(), "SSH command element should have been found")
		Expect(sshCommand).ToNot(BeNil(), "SSH command element should have been found in the previous test")
		command, err := sshCommand.FindElement(selenium.ByCSSSelector, "input[type='text']")
		Expect(err).ToNot(HaveOccurred(), "SSH command input should have been found")
		commandTxt, err := command.GetAttribute("value")
		Expect(err).ToNot(HaveOccurred(), "SSH command text should have been retrieved")
		passwd, err := sshCommand.FindElement(selenium.ByCSSSelector, "input[type='password']")
		Expect(err).ToNot(HaveOccurred(), "SSH password input should have been found")
		passwdTxt, err := passwd.GetAttribute("value")
		Expect(err).ToNot(HaveOccurred(), "SSH password text should have been retrieved")

		Expect(commandTxt).ToNot(BeEmpty(), "SSH command should not be empty")
		Expect(passwdTxt).ToNot(BeEmpty(), "SSH password should not be empty")
		log.Infof("SSH command: %s", commandTxt)

		khRE := regexp.MustCompile(`echo\s+'([^ ]+) ([^ ]+)'`)
		khMatch := khRE.FindStringSubmatch(commandTxt)
		Expect(khMatch).To(HaveLen(2), "Failed to extract known-hosts line")
		hostKeyType := khMatch[1]
		hostKeyData := khMatch[2]

		re := regexp.MustCompile(`(\S+@\S+(?::\d+)?)`)
		matches := re.FindStringSubmatch(commandTxt)
		Expect(matches).To(HaveLen(2), "Failed to parse user@host from SSH command")

		userHost := matches[1] // "user@ip[:port]"
		parts := strings.Split(userHost, "@")
		Expect(parts).To(HaveLen(2))
		username := parts[0]
		hostPort := parts[1] // "ip[:port]"

		host, port, err := net.SplitHostPort(hostPort)
		if err != nil { // no explicit port in string
			host = hostPort
			port = "22"
		}

		config := &ssh.ClientConfig{
			User: username,
			Auth: []ssh.AuthMethod{ssh.Password(passwdTxt)},
			HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
				if key.Type() != hostKeyType {
					return fmt.Errorf("unexpected host key type %s, expected %s", key.Type(), hostKeyType)
				}
				if string(key.Marshal()) != hostKeyData {
					return fmt.Errorf("host key data does not match expected value")
				}
				return nil
			},
			Timeout: 30 * time.Second,
		}

		conn, err := ssh.Dial("tcp", net.JoinHostPort(host, port), config)
		Expect(err).ToNot(HaveOccurred(), "SSH dial failed")
		defer conn.Close()

		sess, err := conn.NewSession()
		Expect(err).ToNot(HaveOccurred(), "Creating SSH session failed")
		defer sess.Close()

		var out bytes.Buffer
		sess.Stdout = &out
		err = sess.Run("uname")
		Expect(err).ToNot(HaveOccurred(), "Failed to send command over SSH")

		Expect(strings.TrimSpace(out.String())).To(Equal("Linux"), "Expected uname output to be Linux")
	})

	It("Should be able to navigate to other regions", func() {
		NUMBER_OF_REGIONS := 41
		err := wd.WaitWithTimeout(conditions.ElementIsLocated(selenium.ByID, "RegionNavButton"), time.Second*30)
		Expect(err).ToNot(HaveOccurred())

		button, err := wd.FindElement(selenium.ByID, "RegionNavButton")
		Expect(err).ToNot(HaveOccurred())

		button.Click()

		panel, err := wd.FindElement(selenium.ByID, "RegionsPanel")
		Expect(err).ToNot(HaveOccurred())

		regionList, err := panel.FindElement(selenium.ByTagName, "ul")
		Expect(err).ToNot(HaveOccurred())

		regions, err := regionList.FindElements(selenium.ByTagName, "li")
		Expect(err).ToNot(HaveOccurred())
		Expect(regions).To(HaveLen(NUMBER_OF_REGIONS))

		for _, region := range regions {
			link, err := region.FindElement(selenium.ByTagName, "a")
			Expect(err).ToNot(HaveOccurred())
			Expect(link.GetAttribute("href")).To(MatchRegexp(`https://([a-z]|[0-9])+\.admin\.aro\.azure\.com`))
		}
	})

	It("Should display the correct cluster detail view for the resource ID parameter in the URL", func() {
		wd.Get(host + resourceIDFromEnv())
		wd.WaitWithTimeout(conditions.ElementIsLocated(selenium.ByID, "ClusterDetailPanel"), time.Second*3)

		detailPanel, err := wd.FindElement(selenium.ByID, "ClusterDetailPanel")
		Expect(err).ToNot(HaveOccurred())
		Expect(detailPanel.IsDisplayed()).To(BeTrue())

		elem, err := wd.FindElement(selenium.ByID, "ClusterDetailName")
		Expect(err).ToNot(HaveOccurred())
		Expect(elem.Text()).To(Equal(clusterName))
	})

	It("Should update URL for each tab in cluster detail page", func() {
		wd.Get(host + resourceIDFromEnv())
		wd.WaitWithTimeout(conditions.ElementIsLocated(selenium.ByID, "ClusterDetailPanel"), time.Second*3)

		detailPanel, err := wd.FindElement(selenium.ByID, "ClusterDetailPanel")
		Expect(err).ToNot(HaveOccurred())
		Expect(detailPanel.IsDisplayed()).To(BeTrue())

		for _, tab := range clusterDetailTabs {
			button, err := wd.FindElement(selenium.ByCSSSelector, fmt.Sprintf("div[name='%s']", tab))
			Expect(err).ToNot(HaveOccurred())
			button.Click()

			currentUrl, err := wd.CurrentURL()
			Expect(err).ToNot(HaveOccurred())
			Expect(strings.ToLower(currentUrl)).To(HaveSuffix("%s%s/%s", host, strings.ToLower(resourceIDFromEnv()), strings.ToLower(tab)))
		}
	})

	It("Should display refresh button to get latest details for each tab in cluster detail page", Pending, func() {
		wd.Get(host + resourceIDFromEnv())
		wd.WaitWithTimeout(conditions.ElementIsLocated(selenium.ByID, "ClusterDetailPanel"), time.Second*3)

		detailPanel, err := wd.FindElement(selenium.ByID, "ClusterDetailPanel")
		Expect(err).ToNot(HaveOccurred())
		Expect(detailPanel.IsDisplayed()).To(BeTrue())

		wd.Wait(conditions.ElementIsLocated(selenium.ByCSSSelector, "div[aria-label='Refresh']"))

		// Check refresh button clicked event for Overview Tab
		button, err := wd.FindElement(selenium.ByCSSSelector, "div[aria-label='Refresh']")
		Expect(err).ToNot(HaveOccurred())
		err = button.Click()
		Expect(err).ToNot(HaveOccurred())
		err = wd.WaitWithTimeout(noContentIsLoading, 2*time.Minute)
		Expect(err).ToNot(HaveOccurred())

		// Check refresh button clicked event for Nodes Tab
		wd.Wait(conditions.ElementIsLocated(selenium.ByCSSSelector, "div[name='Nodes']"))
		nodesButton, err := wd.FindElement(selenium.ByCSSSelector, "div[name='Nodes']")
		Expect(err).ToNot(HaveOccurred())
		nodesButton.Click()

		button, err = wd.FindElement(selenium.ByCSSSelector, "div[aria-label='Refresh']")
		Expect(err).ToNot(HaveOccurred())
		err = button.Click()
		Expect(err).ToNot(HaveOccurred())
		err = wd.WaitWithTimeout(conditions.ElementIsLocated(selenium.ByCSSSelector, "div[role='presentation']"), 2*time.Minute)
		Expect(err).ToNot(HaveOccurred())

		// Check refresh button clicked event for Machines Tab
		wd.Wait(conditions.ElementIsLocated(selenium.ByCSSSelector, "div[name='Machines']"))
		machinesButton, err := wd.FindElement(selenium.ByCSSSelector, "div[name='Machines']")
		Expect(err).ToNot(HaveOccurred())
		machinesButton.Click()

		button, err = wd.FindElement(selenium.ByCSSSelector, "div[aria-label='Refresh']")
		Expect(err).ToNot(HaveOccurred())
		err = button.Click()
		Expect(err).ToNot(HaveOccurred())
		err = wd.WaitWithTimeout(conditions.ElementIsLocated(selenium.ByCSSSelector, "div[role='presentation']"), 2*time.Minute)
		Expect(err).ToNot(HaveOccurred())

		// Check refresh button clicked event for Machine Sets Tab
		wd.Wait(conditions.ElementIsLocated(selenium.ByCSSSelector, "div[name='MachineSets']"))
		machineSetsButton, err := wd.FindElement(selenium.ByCSSSelector, "div[name='MachineSets']")
		Expect(err).ToNot(HaveOccurred())
		machineSetsButton.Click()

		button, err = wd.FindElement(selenium.ByCSSSelector, "div[aria-label='Refresh']")
		Expect(err).ToNot(HaveOccurred())
		err = button.Click()
		Expect(err).ToNot(HaveOccurred())
		err = wd.WaitWithTimeout(conditions.ElementIsLocated(selenium.ByCSSSelector, "div[role='presentation']"), 2*time.Minute)
		Expect(err).ToNot(HaveOccurred())

		// Check refresh button clicked event for API Statistics Tab
		wd.Wait(conditions.ElementIsLocated(selenium.ByCSSSelector, "div[name='APIStatistics']"))
		apiStatisticsButton, err := wd.FindElement(selenium.ByCSSSelector, "div[name='APIStatistics']")
		Expect(err).ToNot(HaveOccurred())
		apiStatisticsButton.Click()

		button, err = wd.FindElement(selenium.ByCSSSelector, "div[aria-label='Refresh']")
		Expect(err).ToNot(HaveOccurred())
		err = button.Click()
		Expect(err).ToNot(HaveOccurred())
		err = wd.WaitWithTimeout(conditions.ElementIsLocated(selenium.ByCSSSelector, "div[role='presentation']"), 2*time.Minute)
		Expect(err).ToNot(HaveOccurred())

		// Check refresh button clicked event for KCM Statistics Tab
		wd.Wait(conditions.ElementIsLocated(selenium.ByCSSSelector, "div[name='KCMStatistics']"))
		kcmStatisticsButton, err := wd.FindElement(selenium.ByCSSSelector, "div[name='KCMStatistics']")
		Expect(err).ToNot(HaveOccurred())
		kcmStatisticsButton.Click()

		button, err = wd.FindElement(selenium.ByCSSSelector, "div[aria-label='Refresh']")
		Expect(err).ToNot(HaveOccurred())
		err = button.Click()
		Expect(err).ToNot(HaveOccurred())
		err = wd.WaitWithTimeout(conditions.ElementIsLocated(selenium.ByCSSSelector, "div[role='presentation']"), 2*time.Minute)
		Expect(err).ToNot(HaveOccurred())

		// Check refresh button clicked event for DNS Statistics Tab
		wd.Wait(conditions.ElementIsLocated(selenium.ByCSSSelector, "div[name='DNSStatistics']"))
		dnsStatisticsButton, err := wd.FindElement(selenium.ByCSSSelector, "div[name='DNSStatistics']")
		Expect(err).ToNot(HaveOccurred())
		dnsStatisticsButton.Click()

		button, err = wd.FindElement(selenium.ByCSSSelector, "div[aria-label='Refresh']")
		Expect(err).ToNot(HaveOccurred())
		err = button.Click()
		Expect(err).ToNot(HaveOccurred())
		err = wd.WaitWithTimeout(conditions.ElementIsLocated(selenium.ByCSSSelector, "div[role='presentation']"), 2*time.Minute)
		Expect(err).ToNot(HaveOccurred())

		// Check refresh button clicked event for Ingress Statistics Tab
		wd.Wait(conditions.ElementIsLocated(selenium.ByCSSSelector, "div[name='IngressStatistics']"))
		ingressStatisticsButton, err := wd.FindElement(selenium.ByCSSSelector, "div[name='IngressStatistics']")
		Expect(err).ToNot(HaveOccurred())
		ingressStatisticsButton.Click()

		button, err = wd.FindElement(selenium.ByCSSSelector, "div[aria-label='Refresh']")
		Expect(err).ToNot(HaveOccurred())
		err = button.Click()
		Expect(err).ToNot(HaveOccurred())
		err = wd.WaitWithTimeout(conditions.ElementIsLocated(selenium.ByCSSSelector, "div[role='presentation']"), 2*time.Minute)
		Expect(err).ToNot(HaveOccurred())

		// Check refresh button clicked event for Cluster Operators Tab
		wd.Wait(conditions.ElementIsLocated(selenium.ByCSSSelector, "div[name='ClusterOperators']"))
		clusterOperatorsButton, err := wd.FindElement(selenium.ByCSSSelector, "div[name='ClusterOperators']")
		Expect(err).ToNot(HaveOccurred())
		clusterOperatorsButton.Click()

		button, err = wd.FindElement(selenium.ByCSSSelector, "div[aria-label='Refresh']")
		Expect(err).ToNot(HaveOccurred())
		err = button.Click()
		Expect(err).ToNot(HaveOccurred())
		err = wd.WaitWithTimeout(conditions.ElementIsLocated(selenium.ByCSSSelector, "div[role='presentation']"), 2*time.Minute)
		Expect(err).ToNot(HaveOccurred())
	})

	It("Should display the action icons on cluster detail page", func() {
		wd.Get(host + resourceIDFromEnv())
		wd.WaitWithTimeout(conditions.ElementIsLocated(selenium.ByID, "ClusterDetailPanel"), time.Second*3)

		detailPanel, err := wd.FindElement(selenium.ByID, "ClusterDetailPanel")
		Expect(err).ToNot(HaveOccurred())
		Expect(detailPanel.IsDisplayed()).To(BeTrue())

		resourceButton, err := wd.FindElement(selenium.ByCSSSelector, "button[aria-label='Copy Resource ID']")
		Expect(err).ToNot(HaveOccurred())
		Expect(resourceButton.IsDisplayed()).To(BeTrue())

		prometheusButton, err := wd.FindElement(selenium.ByCSSSelector, "a[aria-label='Prometheus']")
		Expect(err).ToNot(HaveOccurred())
		Expect(prometheusButton.IsDisplayed()).To(BeTrue())

		sshbutton, err := wd.FindElement(selenium.ByCSSSelector, "button[aria-label='SSH']")
		Expect(err).ToNot(HaveOccurred())
		Expect(sshbutton.IsDisplayed()).To(BeTrue())

		kubeconfigButton, err := wd.FindElement(selenium.ByCSSSelector, "button[aria-label='Download Kubeconfig']")
		Expect(err).ToNot(HaveOccurred())
		Expect(kubeconfigButton.IsDisplayed()).To(BeTrue())
	})
})

func noContentIsLoading(wd selenium.WebDriver) (bool, error) {
	shimmerElements, err := wd.FindElements(selenium.ByCSSSelector, ".ms-Shimmer-container")
	return len(shimmerElements) == 0, err
}
