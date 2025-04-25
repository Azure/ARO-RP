import { IPanelStyles, Panel, PanelType } from "@fluentui/react/lib/Panel"
import { useBoolean } from "@fluentui/react-hooks"
import { useState, useEffect, useRef, MutableRefObject, ReactElement, useMemo } from "react"
import {
  IMessageBarStyles,
  MessageBar,
  MessageBarType,
  Stack,
  Separator,
  IStackStyles,
  Icon,
  IconButton,
  IIconStyles,
} from "@fluentui/react"
import { fetchClusterInfo } from "./Request"
import { IClusterCoordinates, headerStyles } from "./App"
import { Nav, INavLink, INavStyles } from "@fluentui/react/lib/Nav"
import { ToolIcons } from "./ToolIcons"
import { MemoisedClusterDetailListComponent } from "./ClusterDetailList"
import React from "react"
import { useLinkClickHandler, useNavigate, useParams } from "react-router"

const navStyles: Partial<INavStyles> = {
  root: {
    width: 155,
    paddingRight: "10px",
  },
  link: {
    whiteSpace: "normal",
    lineHeight: "inherit",
  },
  groupContent: {
    marginBottom: "0px",
  },
}

const headerStyle: Partial<IStackStyles> = {
  root: {
    alignSelf: "flex-start",
    flexGrow: 2,
    height: 48,
    paddingLeft: 30,
    marginBottom: 15,
  },
}

const doubleChevronIconStyle: Partial<IStackStyles> = {
  root: {
    marginLeft: -30,
    marginTop: -15,
    height: "100%",
    width: "100%",
  },
}

const headerIconStyles: Partial<IIconStyles> = {
  root: {
    height: "100%",
    width: 40,
    paddingTop: 4,
    paddingRight: 10,
    svg: {
      fill: "#e3222f",
    },
  },
}

export const overviewKey = "overview"
export const nodesKey = "nodes"
export const machinesKey = "machines"
export const machineSetsKey = "machinesets"
export const apiStatisticsKey = "apistatistics"
export const kcmStatisticsKey = "kcmstatistics"
export const dnsStatisticsKey = "dnsstatistics"
export const ingressStatisticsKey = "ingressstatistics"
export const clusterOperatorsKey = "clusteroperators"

const errorBarStyles: Partial<IMessageBarStyles> = { root: { marginBottom: 15 } }

export function ClusterDetailPanel(props: {
  csrfToken: MutableRefObject<string>
  sshBox: any
  onClose: any
  loaded: string
}) {
  const [data, setData] = useState<any>([])
  const [error, setError] = useState<Response | null>(null)
  const [fetching, setFetching] = useState("")
  const [isOpen, { setTrue: openPanel, setFalse: dismissPanel }] = useBoolean(false) // panel controls
  const [dataLoaded, setDataLoaded] = useState<boolean>(false)
  const [customPanelStyle, setcustomPanelStyle] = useState<Partial<IPanelStyles>>({
    root: { top: "40px", left: "225px" },
    content: { paddingLeft: 30, paddingRight: 5 },
    navigation: {
      justifyContent: "flex-start",
    },
  })
  const onDismiss = useLinkClickHandler("/")
  const navigate = useNavigate()

  const params = useParams()
  const resourceID = useMemo(() => `/subscriptions/${params.subscriptionId}/resourcegroups/${params.resourceGroupName}/providers/microsoft.redhatopenshift/openshiftclusters/${params.resourceName}`, [params.subscriptionId, params.resourceGroupName, params.resourceName])
  const currentCluster = useMemo<IClusterCoordinates | null>(() => {
    if (params.subscriptionId && params.resourceGroupName && params.resourceName) {
      return {
        subscription: params.subscriptionId,
        resourceGroup: params.resourceGroupName,
        name: params.resourceName,
        resourceId: resourceID,
      }
    }
    return null
  }, [params])

  const errorBar = (): any => {
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

  const navLinkGroups = [
    {
      links: [
        {
          name: "Overview",
          key: overviewKey,
          url: `${resourceID}/${overviewKey}`,
          icon: "Info",
        },
        {
          name: "Nodes",
          key: nodesKey,
          url: `${resourceID}/${nodesKey}`,
          icon: "BranchCommit",
        },
        {
          name: "Machines",
          key: machinesKey,
          url: `${resourceID}/${machinesKey}`,
          icon: "ConnectVirtualMachine",
        },
        {
          name: "MachineSets",
          key: machineSetsKey,
          url: `${resourceID}/${machineSetsKey}`,
          icon: "BuildQueue",
        },
        {
          name: "APIStatistics",
          key: apiStatisticsKey,
          url: `${resourceID}/${apiStatisticsKey}`,
          icon: "BIDashboard",
        },
        {
          name: "KCMStatistics",
          key: kcmStatisticsKey,
          url: `${resourceID}/${kcmStatisticsKey}`,
          icon: "BIDashboard",
        },
        {
          name: "DNSStatistics",
          key: dnsStatisticsKey,
          url: `${resourceID}/${dnsStatisticsKey}`,
          icon: "BIDashboard",
        },
        {
          name: "IngressStatistics",
          key: ingressStatisticsKey,
          url: `${resourceID}/${ingressStatisticsKey}`,
          icon: "BIDashboard",
        },
        {
          name: "ClusterOperators",
          key: clusterOperatorsKey,
          url: `${resourceID}/${clusterOperatorsKey}`,
          icon: "Shapes",
        },
      ],
    },
  ]

  // updateData - updates the state of the component
  // can be used if we want a refresh button.
  // api/clusterdetail returns a single item.
  const updateData = (newData: any) => {
    setData(newData)
  }

  const _dismissPanel = (ev: any) => {
    dismissPanel()
    props.sshBox.current.hidePopup()
    props.onClose()
    setData([])
    setFetching("")
    setDataLoaded(false)
    setError(null)
    onDismiss(ev)
  }

  useEffect(() => {
    if (currentCluster == null) {
      return
    }
    const resourceID = currentCluster.resourceId

    const onData = async (result: Response) => {
      if (result.status === 200) {
        updateData(await result.json())
        setDataLoaded(true)
      } else {
        setError(result)
      }
      setFetching(resourceID)
    }

    if (fetching === "" && props.loaded === "DONE" && resourceID != "") {
      setFetching("FETCHING")
      setError(null)
      fetchClusterInfo(currentCluster).then(onData)
    }
  }, [data, fetching, setFetching])

  useEffect(() => {
    if (currentCluster == null) {
      setDataLoaded(false)
      return
    }
    const resourceID = currentCluster.resourceId

    if (resourceID != "") {
      if (resourceID == fetching) {
        openPanel()
        setDataLoaded(true)
      } else {
        setData([])
        setFetching("")
        setDataLoaded(false) // activate shimmer
        openPanel()
      }
    }
  }, [currentCluster])

  function _onLinkClick(ev?: React.MouseEvent<HTMLElement>, item?: INavLink) {
    if (item && item.name !== "") {
      event?.preventDefault()
      navigate(item.url)
    }
  }

  // const [doubleChevronIconProp, setdoubleChevronIconProp] = useState({ iconName: "doublechevronleft"})
  const doubleChevronIconProp = useRef({ iconName: "doublechevronleft" })
  const _onClickDoubleChevronIcon = () => {
    let customPanelStyleRootLeft
    if (doubleChevronIconProp.current.iconName == "doublechevronright") {
      customPanelStyleRootLeft = "225px"
      // setdoubleChevronIconProp({ iconName: "doublechevronleft"})
      doubleChevronIconProp.current = { iconName: "doublechevronleft" }
    } else {
      customPanelStyleRootLeft = "0px"
      // setdoubleChevronIconProp({ iconName: "doublechevronright"})
      doubleChevronIconProp.current = { iconName: "doublechevronright" }
    }

    setcustomPanelStyle({
      root: { top: "40px", left: customPanelStyleRootLeft },
      content: { paddingLeft: 30, paddingRight: 5 },
      navigation: {
        justifyContent: "flex-start",
      },
    })
  }

  const onRenderHeader = (): ReactElement => {
    return (
      <>
        <Stack styles={headerStyle} horizontal>
          <Stack.Item styles={doubleChevronIconStyle}>
            <IconButton
              onClick={_onClickDoubleChevronIcon}
              iconProps={doubleChevronIconProp.current}
            />
          </Stack.Item>
        </Stack>

        <Stack styles={headerStyle} horizontal>
          <Stack.Item>
            <Icon styles={headerIconStyles} iconName="openshift-svg"></Icon>
          </Stack.Item>
          <Stack.Item>
            <div id="ClusterDetailName" className={headerStyles.titleText}>
              {currentCluster?.name}
            </div>
            <div className={headerStyles.subtitleText}>Cluster</div>
            <ToolIcons
              resourceId={currentCluster ? currentCluster?.resourceId : ""}
              version={Number(data?.version) !== undefined ? Number(data?.version) : 0}
              csrfToken={props.csrfToken}
              sshBox={props.sshBox}
            />
          </Stack.Item>
        </Stack>
      </>
    )
  }

  return (
    <Panel
      id="ClusterDetailPanel"
      isOpen={isOpen}
      type={PanelType.custom}
      onDismiss={_dismissPanel}
      isBlocking={false}
      styles={customPanelStyle}
      closeButtonAriaLabel="Close"
      onRenderHeader={onRenderHeader}>
      <Stack>
        <Stack.Item grow>{error && errorBar()}</Stack.Item>
        <Stack horizontal>
          <Stack.Item>
            <Nav
              onLinkClick={_onLinkClick}
              ariaLabel="Select a tab to view"
              styles={navStyles}
              selectedKey={params["*"]}
              groups={navLinkGroups}
            />
          </Stack.Item>
          <Separator vertical />
          <Stack.Item grow>
            <MemoisedClusterDetailListComponent
              item={data}
              cluster={currentCluster}
              isDataLoaded={dataLoaded}
            />
          </Stack.Item>
        </Stack>
      </Stack>
    </Panel>
  )
}
