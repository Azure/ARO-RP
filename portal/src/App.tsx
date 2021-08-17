import { useState, useEffect, useRef, useCallback } from "react"
import {
  Stack,
  Text,
  Panel,
  IStackTokens,
  IconButton,
  IStackStyles,
  DefaultPalette,
  ThemeProvider,
  PartialTheme,
  PanelType,
  IButtonStyles,
  IPanelProps,
  MessageBar,
  IRenderFunction,
  ITextStyles,
  IPanelStyles,
  TooltipHost,
  IMessageBarStyles,
  MessageBarType,
  Icon,
} from "@fluentui/react"
import { AxiosResponse } from "axios"
import { useBoolean } from "@fluentui/react-hooks"
import { SSHModal } from "./SSHModal"
import { ClusterDetailPanel } from "./ClusterDetail"
import { ClusterList } from "./ClusterList"
import { FetchInfo, ProcessLogOut } from "./Request"

const containerStackTokens: IStackTokens = {}
const appStackTokens: IStackTokens = { childrenGap: 10 }

const errorBarStyles: Partial<IMessageBarStyles> = { root: { marginBottom: 15 } }

const stackStyles: IStackStyles = {
  root: [
    {
      width: "100%",
      padding: 0,
    },
  ],
}

const headerTextStyles: ITextStyles = {
  root: {
    fontWeight: 600,
  },
}

const contentStackStyles: IStackStyles = {
  root: [
    {
      padding: 20,
    },
  ],
}

const stackNavStyles: IStackStyles = {
  root: {
    padding: "0px 15px",
    height: 40,
  },
}

const MenuButtonStyles: IButtonStyles = {
  icon: { color: DefaultPalette.white },
}

const darkTheme: PartialTheme = {
  semanticColors: {
    bodyBackground: DefaultPalette.themePrimary,
    bodyText: DefaultPalette.white,
  },
}

const navPanelStyles: Partial<IPanelStyles> = {
  navigation: {
    height: 40,
    lineHeight: 40,
    fontSize: 15,
    paddingLeft: 15,
    justifyContent: "start",
    alignItems: "center",
  },
}

function App() {
  const [data, updateData] = useState({ location: "", csrf: "", elevated: false, username: "" })
  const [error, setError] = useState<AxiosResponse | null>(null)
  const [isOpen, { setTrue: openPanel, setFalse: dismissPanel }] = useBoolean(false)
  const [fetching, setFetching] = useState("")

  const sshRef = useRef<typeof SSHModal | null>(null)
  const clusterDetailPanelRef = useRef<typeof ClusterDetailPanel | null>(null)
  const csrfRef = useRef<string>("")

  useEffect(() => {
    const onData = (result: AxiosResponse | null) => {
      if (result?.status === 200) {
        updateData(result.data)
        csrfRef.current = result.data.csrf
      } else {
        setError(result)
      }
      setFetching("DONE")
    }

    if (fetching === "") {
      setFetching("FETCHING")
      FetchInfo().then(onData)
    }
  }, [fetching, error, data])

  const onRenderNavigationContent: IRenderFunction<IPanelProps> = useCallback(
    (props, defaultRender) => (
      <>
        <IconButton iconProps={{ iconName: "GlobalNavButton" }} onClick={dismissPanel} />
      </>
    ),
    [dismissPanel]
  )

  const logOut = () => {
    ProcessLogOut()
  }

  const errorBar = (): any => {
    console.log(error)
    return (
      <MessageBar
        messageBarType={MessageBarType.error}
        isMultiline={false}
        onDismiss={() => setError(null)}
        dismissButtonAriaLabel="Close"
        styles={errorBarStyles}
      >
        {error?.statusText}
      </MessageBar>
    )
  }

  return (
    <>
      <Stack styles={stackStyles} tokens={containerStackTokens} horizontalAlign={"stretch"}>
        <Panel
          isLightDismiss
          styles={navPanelStyles}
          type={PanelType.smallFixedNear}
          isOpen={isOpen}
          onDismiss={dismissPanel}
          closeButtonAriaLabel="Close"
          onRenderNavigationContent={onRenderNavigationContent}
        >
          <p>regions go here</p>
        </Panel>
        <ThemeProvider theme={darkTheme}>
          <Stack
            grow
            tokens={appStackTokens}
            horizontalAlign={"start"}
            verticalAlign={"center"}
            horizontal
            styles={stackNavStyles}
          >
            <Stack.Item>
              <IconButton
                iconProps={{ iconName: "GlobalNavButton" }}
                onClick={openPanel}
                styles={MenuButtonStyles}
              />
            </Stack.Item>
            <Stack.Item grow>
              <Text styles={headerTextStyles}>
                ARO Portal {data.location ? "(" + data.location + ")" : ""}
              </Text>
            </Stack.Item>
            <Stack.Item>
              <Text>{data.username}</Text>
            </Stack.Item>

            <Stack.Item hidden={!data.elevated}>
              <TooltipHost content={`Elevated User`}>
                <Icon iconName={"Admin"}></Icon>
              </TooltipHost>
            </Stack.Item>
            <Stack.Item>
              <IconButton
                iconProps={{ iconName: "SignOut" }}
                onClick={logOut}
                styles={MenuButtonStyles}
              />
            </Stack.Item>
          </Stack>
        </ThemeProvider>
        <Stack styles={contentStackStyles}>
          <Stack.Item grow>{error && errorBar()}</Stack.Item>
          <Stack.Item grow>
            <ClusterList csrfToken={csrfRef} sshBox={sshRef} clusterDetailPanel={clusterDetailPanelRef} loaded={fetching} />
          </Stack.Item>
        </Stack>
        <SSHModal csrfToken={csrfRef} ref={sshRef} />
        <ClusterDetailPanel csrfToken={csrfRef} loaded={fetching} name={""} resourceGroup={""} subscription={""} ref={clusterDetailPanelRef} />
        {/* ClusterDetailPanel should be moved into ClusterList 
        --- loaded is a prop that stops the panel loading further data until 
        main api has fetched api/info 
        
        when we move this component in... ref to error bar in clusterlist.. ? 
        ... maybe not - no., actual not.. we want the error to appear in panel.
        */}
      </Stack>
    </>
  )
}

export default App
