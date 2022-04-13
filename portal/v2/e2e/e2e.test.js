const { By, Builder, Key } = require("selenium-webdriver");
const edge = require("selenium-webdriver/edge");
const { until } = require("selenium-webdriver");

const getEdgeOptions = () => {
  let edgeOptions = new edge.Options
  edgeOptions.setAcceptInsecureCerts(true)
  return edgeOptions;
};

const getEdgeDriver = () => {
  return new Builder()
  .forBrowser('MicrosoftEdge')
  .usingServer("http://127.0.0.1:4444/wd/hub/")
  .setEdgeService(
      edge.setDefaultService(
          new edge.ServiceBuilder("/bin/msedgedriver").addArguments()
      )
  ).setEdgeOptions(getEdgeOptions())
  .build();
};

const driver = getEdgeDriver()

beforeAll(async () => {
  // Setup session cookie
  await driver.get("https://127.0.0.1:8444/api/info")
  await driver.manage().addCookie({name:'session', value: 'MTY0OTgzMjAxOXxEdi1CQkFFQ180SUFBUkFCRUFBQVRmLUNBQU1HYzNSeWFXNW5EQXNBQ1hWelpYSmZibUZ0WlFaemRISnBibWNNQmdBRWRHVnpkQVp6ZEhKcGJtY01DQUFHWjNKdmRYQnpDRnRkYzNSeWFXNW5fNE1DQVFMX2hBQUJEQUFBVnYtRUp3QUJKRGt6TkRabE9EWXpMVEE0TXprdE5ESTRNQzA0WmpKbExUY3hZMk0xWkRKaU1ESXpOQVp6ZEhKcGJtY01DUUFIWlhod2FYSmxjd2wwYVcxbExsUnBiV1hfaFFVQkFRUlVhVzFsQWYtR0FBQUFGUC1HRVFBUEFRQUFBQTdaNkhWakF0MFVkd0pZfNKlEuJDXgv1sgWoBdl-1z-ZTuLvQlZAxOF9CJRmWtos'});

  await driver.get("https://127.0.0.1:8444/v2")
});

afterAll(async () => {
  await driver.quit();
});

beforeEach(async () => {
  await driver.get("https://127.0.0.1:8444/v2");
  await driver.navigate().refresh();
});

describe("Admin Portal E2E Testing", () => {
  jest.setTimeout(60000);

  test('Check cluster data populates correctly', async () => {
    const cluster = await driver.wait(until.elementLocated(By.css("div[data-automation-key='name']")), 10000).getText();
    expect(cluster).toEqual("elljohns-test");
  })

  test('Check cluster data filter works correctly', async () => {
    const filter = await driver.wait(until.elementLocated(By.css("input[placeholder='Filter on resource ID']")), 10000);
    await driver.wait(until.elementLocated(By.css("div[data-automation-key='name']")), 10000);
    filter.sendKeys("elljohns-admin-portal-testing")

    await driver.wait(until.elementLocated(By.css("div[data-automation-key='name']")), 10000);
    await driver.wait(until.elementTextIs(driver.findElement(By.css("div[data-automation-key='name']")), "elljohns-admin-portal-testing"), 10000);

    const cluster = await driver.wait(until.elementLocated(By.css("div[data-automation-key='name']")), 10000).getText();
    expect(cluster).toEqual("elljohns-admin-portal-testing");

    await filter.clear()
  })

  test('Check cluster Info Panel Populates Correctly', async () => {
    const testValues = ["Public",
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
                        "Installed"]

    const cluster = await driver.wait(until.elementLocated(By.css("div[data-automation-key='name']")), 10000);
    await driver.wait(until.elementIsVisible(cluster), 10000);

    cluster.click();

    const panel = await driver.wait(until.elementLocated(By.className("ms-Panel is-open ms-Panel--hasCloseButton ms-Panel--custom root-225")), 10000);
    await driver.wait(until.elementIsVisible(panel), 10000);

    const panelfields = await driver.wait(until.elementsLocated(By.className("css-287")), 10000)

    for( var i = 0; i < panelfields.length; i++){   
      const panelText = await panelfields[i].getText()
      if ( panelText === ":") { 
        panelfields.splice(i, 1);
        i--; 
      }
    }

    const panelvalues = await driver.wait(until.elementsLocated(By.className("css-290")), 10000)

    for( var i = 0; i < panelvalues.length; i++){   
      const panelFieldText = await panelfields[i].getText()
      const panelValueText = await panelvalues[i].getText()

      expect(panelFieldText +  " : " + panelValueText).toEqual(panelFieldText + " : " + testValues[i])
    }
  })

  // test('Check cluster Resource Id Copy works', async () => {
  //   const cluster = await driver.wait(until.elementLocated(By.css("div[data-automation-key='name']")), 10000).getText();
  //   expect(cluster).toEqual("elljohns-test");
  // })

  // test('Check cluster kubeconfig download works', async () => {
  //   const cluster = await driver.wait(until.elementLocated(By.css("div[data-automation-key='name']")), 10000).getText();
  //   expect(cluster).toEqual("elljohns-test");
  // })

  // test('Check cluster ssh details works', async () => {
  //   const cluster = await driver.wait(until.elementLocated(By.css("div[data-automation-key='name']")), 10000).getText();
  //   expect(cluster).toEqual("elljohns-test");
  // })
});

