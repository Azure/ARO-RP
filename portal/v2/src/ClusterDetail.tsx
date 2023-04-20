import {
  IPanelStyles,
  Panel,
  PanelType,
} from "@fluentui/react/lib/Panel"
import { useBoolean } from "@fluentui/react-hooks"
import { useState, useEffect, useRef, MutableRefObject, ReactElement } from "react"
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
import { AxiosResponse } from "axios"
import { FetchClusterInfo } from "./Request"
import { ICluster, headerStyles } from "./App"
import { Nav, INavLink, INavStyles } from "@fluentui/react/lib/Nav"
import { ClusterDetailComponent } from "./ClusterDetailList"
import React from "react"

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


// let customPanelStyle: Partial<IPanelStyles> = {
//   root: { top: "40px", left: "225px" },
//   content: { paddingLeft: 30, paddingRight: 5 },
//   navigation: {
//     justifyContent: "flex-start",
//   },
// }

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

const errorBarStyles: Partial<IMessageBarStyles> = { root: { marginBottom: 15 } }

export function ClusterDetailPanel(props: {
  csrfToken: MutableRefObject<string>
  currentCluster: ICluster | null
  onClose: any
  loaded: string
}) {
  const [data, setData] = useState<any>([])
  const [error, setError] = useState<AxiosResponse | null>(null)
  const state = useRef<ClusterDetailComponent>(null)
  const [fetching, setFetching] = useState("")
  const [isOpen, { setTrue: openPanel, setFalse: dismissPanel }] = useBoolean(false) // panel controls
  const [dataLoaded, setDataLoaded] = useState<boolean>(false)
  const [detailPanelVisible, setdetailPanelVisible] = useState<string>("Overview")
  const [customPanelStyle, setcustomPanelStyle] = useState<Partial<IPanelStyles>>({
    root: { top: "40px", left: "225px" },
    content: { paddingLeft: 30, paddingRight: 5 },
    navigation: {
      justifyContent: "flex-start",
    },
  })

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
          name: 'Overview',
          key: overviewKey,
          url: '#overview',
          icon: 'ThisPC',
        },
        {
          name: 'Nodes',
          key: nodesKey,
          url: '#nodes',
          icon: 'BuildQueue',
        },
        {
          name: 'Machines',
          key: machinesKey,
          url: '#machines',
          icon: 'BuildQueue',
        },
        {
          name: 'MachineSets',
          key: machineSetsKey,
          url: '#machinesets',
          icon: 'BuildQueue',
        },
      ],
    },
  ];

  // updateData - updates the state of the component
  // can be used if we want a refresh button.
  // api/clusterdetail returns a single item.
  const updateData = (newData: any) => {
    setData(newData)
    if (state && state.current) {
      state.current.setState({ item: newData })
    }
  }

  const _dismissPanel = () => {
    dismissPanel()
    props.onClose() // useEffect?
    setData([])
    setFetching("")
    setDataLoaded(false)
    setError(null)
  }

  useEffect(() => {
    if (props.currentCluster == null) {
      return
    }
    const resourceID = props.currentCluster.resourceId

    const onData = (result: AxiosResponse | null) => {
      if (result?.status === 200) {
        updateData(result.data)
        setDataLoaded(true)
      } else {
        setError(result)
      }
      setFetching(resourceID)
    }

    if (fetching === "" && props.loaded === "DONE" && resourceID != "") {
      setFetching("FETCHING")
      setError(null)
      FetchClusterInfo(props.currentCluster).then(onData)
    }
  }, [data, fetching, setFetching])

  useEffect(() => {
    if (props.currentCluster == null) {
      setDataLoaded(false)
      return
    }
    const resourceID = props.currentCluster.resourceId

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
  }, [props.currentCluster?.resourceId])

  function _onLinkClick(ev?: React.MouseEvent<HTMLElement>, item?: INavLink) {
    if (item && item.name !== "") {
      setdetailPanelVisible(item.name)
    }
  }

  const [doubleChevronIconProp, setdoubleChevronIconProp] = useState({ iconName: "doublechevronleft"})
  function _onClickDoubleChevronIcon() {
    let customPanelStyleRootLeft
    if (doubleChevronIconProp.iconName == "doublechevronright") {
      customPanelStyleRootLeft = "225px"
      setdoubleChevronIconProp({ iconName: "doublechevronleft"})
    } else {
      customPanelStyleRootLeft = "0px"
      setdoubleChevronIconProp({ iconName: "doublechevronright"})
    }

    setcustomPanelStyle({
      root: { top: "40px", left: customPanelStyleRootLeft },
      content: { paddingLeft: 30, paddingRight: 5 },
      navigation: {
        justifyContent: "flex-start",
      },
    })
  }


  const onRenderHeader = (
  ): ReactElement => {
    return (
      <>
        <Stack styles={headerStyle} horizontal>
          <Stack.Item styles={doubleChevronIconStyle}>
            <IconButton
              onClick={_onClickDoubleChevronIcon}
              iconProps={doubleChevronIconProp}
            />
          </Stack.Item>
        </Stack>

        <Stack styles={headerStyle} horizontal>
          <Stack.Item>
            <Icon styles={headerIconStyles} iconName="openshift-svg"></Icon>
          </Stack.Item>
          <Stack.Item>
            <div className={headerStyles.titleText}>{props.currentCluster?.name}</div>
            <div className={headerStyles.subtitleText}>Cluster</div>
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
              groups={navLinkGroups}
            />
          </Stack.Item>
          <Separator vertical />
          <Stack.Item grow>
            <ClusterDetailComponent
              item={data}
              cluster={props.currentCluster}
              isDataLoaded={dataLoaded}
              detailPanelVisible={detailPanelVisible}
            />
          </Stack.Item>
        </Stack>
      </Stack>
    </Panel>
  )
}
