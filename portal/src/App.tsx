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
  mergeStyleSets,
  registerIcons,
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

export interface ICluster {
  key: string
  name: string
  subscription: string
  resourceGroup: string
  id: string
  version: string
  createdDate: string
  provisionedBy: string
  provisioningState: string
  failedProvisioningState: string
  resourceId: string
  consoleLink: string
}

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

export const contentStackStylesNormal: IStackStyles = {
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

export const headerStyles = mergeStyleSets({
  titleText: {
    fontWeight: 600,
    fontSize: 24,
    lineHeight: 32,
  },
  subtitleText: {
    color: "#646464",
    fontSize: 12,
    lineHeight: 14,
    margin: 0,
  },
})

registerIcons({
  icons: {
    "openshift-svg": (
      <svg xmlns="http://www.w3.org/2000/svg" width="100%" height="100%" viewBox="0 0 64 64">
        <g>
          <path d="M17.424 27.158L7.8 30.664c.123 1.545.4 3.07.764 4.566l9.15-3.333c-.297-1.547-.403-3.142-.28-4.74M60 16.504c-.672-1.386-1.45-2.726-2.35-3.988l-9.632 3.506c1.12 1.147 2.06 2.435 2.83 3.813z" />
          <path d="M38.802 13.776c2.004.935 3.74 2.21 5.204 3.707l9.633-3.506a27.38 27.38 0 0 0-10.756-8.95c-13.77-6.42-30.198-.442-36.62 13.326a27.38 27.38 0 0 0-2.488 13.771l9.634-3.505c.16-2.087.67-4.18 1.603-6.184 4.173-8.947 14.844-12.83 23.79-8.658" />
        </g>
        <path d="M9.153 35.01L0 38.342c.84 3.337 2.3 6.508 4.304 9.33l9.612-3.5a17.99 17.99 0 0 1-4.763-9.164" />
        <path d="M49.074 31.38a17.64 17.64 0 0 1-1.616 6.186c-4.173 8.947-14.843 12.83-23.79 8.657a17.71 17.71 0 0 1-5.215-3.7l-9.612 3.5c2.662 3.744 6.293 6.874 10.748 8.953 13.77 6.42 30.196.44 36.618-13.328a27.28 27.28 0 0 0 2.479-13.765l-9.61 3.498z" />
        <path d="M51.445 19.618l-9.153 3.332c1.7 3.046 2.503 6.553 2.24 10.08l9.612-3.497c-.275-3.45-1.195-6.817-2.7-9.915" />
      </svg>
    ),
  },
})

export interface IClusterDetail {
  subscription: string
  resourceGroup: string
  clusterName: string
}

function App() {
  const [data, updateData] = useState({ location: "", csrf: "", elevated: false, username: "" })
  const [error, setError] = useState<AxiosResponse | null>(null)
  const [isOpen, { setTrue: openPanel, setFalse: dismissPanel }] = useBoolean(false)
  const [fetching, setFetching] = useState("")
  const [currentCluster, setCurrentCluster] = useState<ICluster | null>(null)

  const [contentStackStyles, setContentStackStyles] =
    useState<IStackStyles>(contentStackStylesNormal)
  const sshRef = useRef<typeof SSHModal | null>(null)
  const csrfRef = useRef<string>("")

  const _onCloseDetailPanel = () => {
    setCurrentCluster(null)
    setContentStackStyles(contentStackStylesNormal)
  }

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
    () => (
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
        styles={errorBarStyles}>
        {error?.statusText}
      </MessageBar>
    )
  }

  // Application state maintains the current resource id/name/group
  // when we click a thing set the state
  // ...

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
          onRenderNavigationContent={onRenderNavigationContent}>
          <p>regions go here</p>
        </Panel>
        <ThemeProvider theme={darkTheme}>
          <Stack
            grow
            tokens={appStackTokens}
            horizontalAlign={"start"}
            verticalAlign={"center"}
            horizontal
            styles={stackNavStyles}>
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
            <ClusterList
              csrfToken={csrfRef}
              sshBox={sshRef}
              setCurrentCluster={setCurrentCluster}
              loaded={fetching}
            />
          </Stack.Item>
          <Stack.Item grow>
            <ClusterDetailPanel
              csrfToken={csrfRef}
              loaded={fetching}
              currentCluster={currentCluster}
              onClose={_onCloseDetailPanel}
            />
          </Stack.Item>
        </Stack>
        <SSHModal csrfToken={csrfRef} ref={sshRef} />
      </Stack>
    </>
  )
}

export default App
