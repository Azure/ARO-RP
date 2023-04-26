import { useState } from "react"
import { ICluster } from "../../App"
import {
  Modal,
  Stack,
  Text,
  IStackStyles,
  IconButton,
  mergeStyleSets,
  getTheme,
  Label,
  ThemeProvider,
  IStackTokens,
  PartialTheme,
  DefaultPalette,
} from "@fluentui/react"
import { useBoolean } from "@fluentui/react-hooks"
import { StatisticsWrapper } from "./StatisticsWrapper"

import { GraphOptionsComponent } from "./GraphOptionsComponent"

export const iconButtonStyles = mergeStyleSets({
  icon: {
    color: "white",
  },
  root: {
    selectors: {
      ":hover .ms-Button-icon": {
        color: DefaultPalette.accent,
      },
    },
  },
})

const global = new Date()
export function Statistics(props: {
  currentCluster: ICluster
  detailPanelSelected: string
  loaded: boolean
  statisticsType: string
}) {
  const [globalDuration, setGlobalDuration] = useState<string>("1h")
  const [globalEndDate, setGlobalEndDate] = useState<Date>(global)

  function GlobalGraphOptionsBar() {
    const stackStyles: IStackStyles = {
      root: [
        {
          width: "100%",
          padding: 0,
        },
      ],
    }
    const stackNavStyles: IStackStyles = {
      root: {
        padding: "0px 15px",
        height: 40,
      },
    }
    const containerStackTokens: IStackTokens = {}
    const appStackTokens: IStackTokens = { childrenGap: 10 }
    return (
      <Stack styles={stackStyles} tokens={containerStackTokens} horizontalAlign={"stretch"}>
        <ThemeProvider theme={darkTheme}>
          <Stack
            grow
            tokens={appStackTokens}
            horizontalAlign={"start"}
            verticalAlign={"center"}
            horizontal
            styles={stackNavStyles}>
            <Stack.Item grow>
              <Text>{"Global Graph Options"}</Text>
            </Stack.Item>
            <Stack.Item>
              <GraphOptionsComponent
                duration={globalDuration}
                setDuration={setGlobalDuration}
                endDate={globalEndDate}
                setEndDate={setGlobalEndDate}
              />
            </Stack.Item>
          </Stack>
        </ThemeProvider>
      </Stack>
    )
  }

  const darkTheme: PartialTheme = {
    semanticColors: {
      bodyBackground: DefaultPalette.accent,
      bodyText: DefaultPalette.white,
    },
    defaultFontStyle: {
      fontWeight: 500,
    },
  }
  const theme = getTheme()

  function GraphWrapper(lprops: { heading: string; statisticsName: string }) {
    const [isModalOpen, { setTrue: showModal, setFalse: hideModal }] = useBoolean(false)
    const [duration, setDuration] = useState<string>(globalDuration)
    const [endDate, setEndDate] = useState<Date>(globalEndDate)
    return (
      <>
        <Modal
          titleAriaId={lprops.heading}
          isOpen={isModalOpen}
          onDismiss={hideModal}
          isBlocking={false}>
          <Stack
            style={{ boxShadow: theme.effects.elevation8 }}
            styles={{ root: { margin: "2px" } }}>
            <ThemeProvider theme={darkTheme}>
              <Stack
                horizontalAlign="stretch"
                horizontal /*className={classNames.iconContainer} /*style={{ boxShadow: theme.effects.elevation64 }}*/
              >
                <Stack.Item grow={0.5}>
                  <GraphOptionsComponent
                    duration={duration}
                    setDuration={setDuration}
                    endDate={endDate}
                    setEndDate={setEndDate}
                  />
                </Stack.Item>
                <Stack.Item align="center" grow={1}>
                  <Label> {lprops.heading} </Label>
                </Stack.Item>
                <Stack.Item align="center">
                  <IconButton
                    iconProps={{ iconName: "Cancel" }}
                    ariaLabel="Close popup modal"
                    onClick={hideModal}
                    styles={iconButtonStyles}
                  />
                </Stack.Item>
              </Stack>
            </ThemeProvider>
            <StatisticsWrapper
              currentCluster={props.currentCluster}
              detailPanelSelected={props.detailPanelSelected}
              loaded={props.loaded}
              statisticsName={lprops.statisticsName}
              duration={duration}
              endDate={endDate}
              graphHeight={500}
              graphWidth={1500}
            />
          </Stack>
        </Modal>
        <Stack style={{ boxShadow: theme.effects.elevation8 }} styles={{ root: { margin: "2px" } }}>
          <ThemeProvider theme={darkTheme}>
            <Stack horizontalAlign="stretch" horizontal>
              <Stack.Item align="center">
                <IconButton
                  onClick={showModal}
                  iconProps={{ iconName: "FullScreen" }}
                  styles={iconButtonStyles}
                />
              </Stack.Item>
              <Stack.Item align="center" grow={1}>
                <Text> {lprops.heading} </Text>
              </Stack.Item>
              <Stack.Item>
                <GraphOptionsComponent
                  duration={duration}
                  setDuration={setDuration}
                  endDate={endDate}
                  setEndDate={setEndDate}
                />
              </Stack.Item>
            </Stack>
          </ThemeProvider>
          <StatisticsWrapper
            currentCluster={props.currentCluster}
            detailPanelSelected={props.detailPanelSelected}
            loaded={props.loaded}
            statisticsName={lprops.statisticsName}
            duration={duration}
            endDate={endDate}
            graphHeight={200}
            graphWidth={740}
          />
        </Stack>
      </>
    )
  }

  interface Map {
    [key: string]: {
      stasticsName: string
      heading: string
    }[]
  }

  let statisticsDataMap: Map = {
    api: [
      { stasticsName: "kubeapicodes", heading: "KubeAPI Server response sizes by code and verb" },
      { stasticsName: "kubeapicpu", heading: "KubeAPI CPU per instance" },
      { stasticsName: "kubeapimemory", heading: "KubeAPI Memory per instance" },
    ],
    kcm: [
      {
        stasticsName: "kubecontrollermanagercodes",
        heading: "Kube Controller Manager Server response sizes by code and verb",
      },
      {
        stasticsName: "kubecontrollermanagercpu",
        heading: "Kube Controller Manager CPU per instance",
      },
      {
        stasticsName: "kubecontrollermanagermemory",
        heading: "Kube Controller Manager Memory per instance",
      },
    ],
    dns: [
      {
        stasticsName: "dnsresponsecodes",
        heading: "Response Codes",
      },
      {
        stasticsName: "dnsalltraffic",
        heading: "All Traffic",
      },
      {
        stasticsName: "dnserrorrate",
        heading: "Error Rate",
      },
      {
        stasticsName: "dnshealthcheck",
        heading: "Health Check",
      },
      {
        stasticsName: "dnsforwardedtraffic",
        heading: "Forwarded Traffic",
      },
    ],
    ingress: [
      {
        stasticsName: "ingresscontrollercondition",
        heading: "Ingress Controller Condition",
      },
    ],
  }

  const statisticsJSX = (sDataMap: Map): JSX.Element[] => {
    let sDataArray = sDataMap[props.statisticsType]
    let sDataArrayLength = sDataArray.length
    let stackItems: JSX.Element[] = []
    let stacks: JSX.Element[] = []
    sDataArray.map((sData, i) => {
      if (i % 2 != 0 || i === sDataArrayLength - 1) {
        stackItems.push(
          <Stack.Item>
            <GraphWrapper statisticsName={sData.stasticsName} heading={sData.heading} />
          </Stack.Item>
        )
        stacks.push(<Stack horizontal>{stackItems}</Stack>)
        stackItems = []
        return
      }
      stackItems.push(
        <Stack.Item>
          <GraphWrapper statisticsName={sData.stasticsName} heading={sData.heading} />
        </Stack.Item>
      )
    })
    return stacks
  }

  return (
    <>
      <GlobalGraphOptionsBar />
      {statisticsJSX(statisticsDataMap)}
    </>
  )
}
