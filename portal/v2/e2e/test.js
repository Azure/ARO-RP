const edge = require("selenium-webdriver/edge");
const { Builder, By, Key, until } = require("selenium-webdriver");
const script = require('jest');



(async function googleSearch() {

  let edgeOptions = new edge.Options
  edgeOptions.setAcceptInsecureCerts(true)
  let driver = await new Builder()
    .forBrowser("MicrosoftEdge")
    .usingServer("http://127.0.0.1:4444/wd/hub/")
    .setEdgeService(
        edge.setDefaultService(
            new edge.ServiceBuilder("/bin/msedgedriver").addArguments()
        )
    ).setEdgeOptions(edgeOptions)
    .build();
  try {
    // Navigate to Url
    await driver.get("https://127.0.0.1:8444");
    // await driver.get("https://www.google.com");
    // Enter text "Automation Bro" and perform keyboard action "Enter"

    await new Promise(r => setTimeout(r, 30000));

    // console.log(await driver.getCurrentUrl);
    console.log(await (await driver.getCapabilities()).getBrowserName());
    console.log(await (await driver.getCapabilities()).getBrowserVersion());
  } finally {
    driver.quit();
  }
})();