import { IPanelStyles, Panel, PanelType } from '@fluentui/react/lib/Panel';
import { useBoolean } from '@fluentui/react-hooks';
import { useState, useEffect, useRef, MutableRefObject } from "react"
import { IMessageBarStyles, MessageBar, MessageBarType, Stack, Separator } from '@fluentui/react';
import { AxiosResponse } from 'axios';
import { FetchClusterInfo } from './Request';
import { IClusterDetail, contentStackStylesNormal } from "./App"
import { Nav, INavLink, INavLinkGroup, INavStyles } from '@fluentui/react/lib/Nav';
import { ClusterDetailComponent } from './ClusterDetailList'

const navStyles: Partial<INavStyles> = {
  root: {
    width: 155,
    paddingRight: "10px"
  },
  link: {
    whiteSpace: 'normal',
    lineHeight: 'inherit',
  },
  groupContent: {
    marginBottom: "0px"
  }
};

const navLinkGroups: INavLinkGroup[] = [
  {
    links: [
      {
        name: 'Overview',
        key: 'overview',
        url: "#overview",
        icon: 'ThisPC',
      },
    ],
  },
  {
    links: [
      {
        name: 'Nodes',
        key: 'nodes',
        url: "#nodes",
        icon: 'BuildQueue'
      },
    ],
  },
];

const customPanelStyle: Partial<IPanelStyles> = {
  root: { top: "40px", left: "225px" },
  content: { paddingLeft: 5, paddingRight: 5, },
}

const errorBarStyles: Partial<IMessageBarStyles> = { root: { marginBottom: 15 } }

export function ClusterDetailPanel(props: {
  csrfToken: MutableRefObject<string>
  currentCluster: IClusterDetail
  onClose: any // TODO: function ptr .. any probably bad
  loaded: string
}) {
  const [data, setData] = useState<any>([])
  const [error, setError] = useState<AxiosResponse | null>(null)
  const state = useRef<ClusterDetailComponent>(null)
  const [fetching, setFetching] = useState("")
  const [resourceID, setResourceID] = useState("")
  const [isOpen, { setTrue: openPanel, setFalse: dismissPanel }] = useBoolean(false); // panel controls
  const [dataLoaded, setDataLoaded] = useState<boolean>(false);
  const [detailPanelVisible, setdetailPanelVisible] = useState<string>("Overview");

  const errorBar = (): any => {
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
    props.currentCluster.clusterName = ""
    props.onClose() // useEffect?
    setDataLoaded(false);
  }

  useEffect(() => {
    const onData = (result: AxiosResponse | null) => {
      if (result?.status === 200) {
        updateData(result.data)
        setDataLoaded(true);
      } else {
        setError(result)
      }
      setFetching(props.currentCluster.clusterName)
    }

    if (fetching === "" && props.loaded === "DONE" && props.currentCluster.clusterName != "") {
      setFetching("FETCHING")
      FetchClusterInfo(props.currentCluster.subscription, props.currentCluster.resource, props.currentCluster.clusterName).then(onData) // TODO: fetchClusterInfo accepts IClusterDetail
    }
  }, [data, fetching, setFetching])


  useEffect(() => {
    if (props.currentCluster.clusterName != "") {
      if (props.currentCluster.clusterName == fetching) {
        openPanel()
        setDataLoaded(true);
      } else {
        setData([])
        setFetching("")
        setDataLoaded(false); // activate shimmer
        openPanel()
      }
    }
  }, [props.currentCluster.clusterName])

  function _onLinkClick(ev?: React.MouseEvent<HTMLElement>, item?: INavLink) {
    if (item && item.name !== '') {
      setdetailPanelVisible(item.name)
    }
  }

  // TODO: props.loaded rename to CSRFTokenAvailable
  return (
    <Panel
      isOpen={isOpen}
      type={PanelType.custom}
      onDismiss={_dismissPanel}
      isBlocking={false}
      styles={customPanelStyle}
      closeButtonAriaLabel="Close"
      headerText={resourceID}
    >
      <Stack styles={contentStackStylesNormal}>
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
              clusterName={props.currentCluster.clusterName}
              isDataLoaded={dataLoaded}
              detailPanelVisible={detailPanelVisible}
            />
          </Stack.Item>
        </Stack>
      </Stack>
    </Panel>
  )
}
