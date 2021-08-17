import { DefaultButton } from '@fluentui/react/lib/Button';
import { IPanelStyles, Panel, PanelType } from '@fluentui/react/lib/Panel';
import { useBoolean } from '@fluentui/react-hooks';
import { Shimmer, ShimmerElementsGroup, ShimmerElementType } from '@fluentui/react/lib/Shimmer';
import React, { useState, useImperativeHandle, useEffect, Component, useRef, forwardRef, MutableRefObject } from "react"
import { DetailsRow, GroupedList, IColumn, IGroup, IMessageBarStyles, MessageBar, MessageBarType, Toggle, Stack } from '@fluentui/react';
//import { createListItems, createGroups, IExampleItem } from '@fluentui/example-data';
import { AxiosResponse } from 'axios';
import { FetchClusterInfo } from './Request';
import { IClusterDetail, contentStackStylesNormal } from "./App"
import { Nav, INavLinkGroup, INavStyles } from '@fluentui/react/lib/Nav';

const navStyles: Partial<INavStyles> = {
  root: {
    width: 155,
    //padding: 5,
  },
  link: {
    whiteSpace: 'normal',
    lineHeight: 'inherit',
  },
};

const navLinkGroups: INavLinkGroup[] = [
  {
    links: [
      {
        name: 'Overview',
        key: 'overview',
        url: "#overview",
        isExpanded: true,
        target: '_blank',
        icon: 'Overview',
      },
    ],
  },
  {
    links: [
      {
        name: 'Nodes',
        key: 'nodes',
        url: "#nodes",
        isExpanded: true,
        target: '_blank',
      },
    ],
  },
  {
    name: "Extra",
    links: [
      {
        name: 'Something',
        key: 'something',
        url: "#something",
        isExpanded: true,
        target: '_blank',
      },
    ],
  },
];

// does the controller need props?
type ClusterDetailPanelProps = {
  csrfToken: MutableRefObject<string>
  name: any
  subscription: any
  resourceGroup: any
  loaded: string
}

interface IClusterDetails {
  apiServer: any
  architectureVersion: string
  consoleLink: string
  createdAt: string
  createdBy: string
  failedProvisioningState: string
  infraId: string
  ingressProfiles: any
  lastAdminUpdateError: string
  lastModifiedAt: string
  lastModifiedBy: string
  lastProvisioningState: string
  location: string
  masterProfile: string
  name: string
  provisioningState: string
  resourceId: string
  tags: any
  version: string
  workerProfile: any
}

interface ClusterDetailComponentProps {
  item: IClusterDetails
  clusterName: string
  isDataLoaded: boolean
}

interface IClusterDetailComponentState {
  item: IClusterDetails // why both state and props?
}

const columns: IColumn[] = [{
  key: "key",
  name: "key",
  fieldName: "key",
  minWidth: 300
},
{
  key: "value",
  name: "value",
  fieldName: "value",
  minWidth: 300
},]
const onRenderCell = (
  nestingDepth?: number,
  item?: any,
  itemIndex?: number,
  group?: IGroup,
): React.ReactNode => {
  return item && typeof itemIndex === 'number' && itemIndex > -1 ? (
    <DetailsRow
      columns={columns}
      groupNestingDepth={nestingDepth}
      item={item}
      itemIndex={itemIndex}
    />
  ) : null;
};


class ClusterDetailComponent extends Component<ClusterDetailComponentProps, IClusterDetailComponentState> {

  constructor(props: ClusterDetailComponentProps | Readonly<ClusterDetailComponentProps>) {
    super(props);
    // this.state = { item: this.props.item };
  }

  public render() {
    return (
      <div>
        <h1>{this.props.clusterName}</h1>
        <Shimmer isDataLoaded={this.props.isDataLoaded}>
          <GroupedList
            onRenderCell={onRenderCell}
            items={Object.keys(this.props.item)}></GroupedList>
          <div>{this.props.item.resourceId}</div>
          <div>{this.props.item.name}</div>
          <div>{this.props.item.lastModifiedAt}</div>
          <div>{this.props.item.lastModifiedBy}</div>
          <div>{this.props.item.lastProvisioningState}</div>
          <div>{this.props.item.provisioningState}</div>
          <div>{this.props.item.location}</div>
          <div>{this.props.item.version}</div>
          <div>{this.props.item.consoleLink}</div>
        </Shimmer>
      </div>
    );
  }
};

const customPanelStyle: Partial<IPanelStyles> = {
  root: { top: "40px", left: "225px"} ,
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


  function _onRenderGroupHeader(group: INavLinkGroup): JSX.Element {
    return <h3>{group.name}</h3>;
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
          <Stack.Item grow>
            <Nav
              //onLinkClick={_onLinkClick}
              selectedKey="key3"
              ariaLabel="Nav basic example"
              styles={navStyles}
              groups={navLinkGroups}
            />
          </Stack.Item>
          <Stack.Item grow>
            <ClusterDetailComponent
              item={data}
              clusterName={props.currentCluster.clusterName}
              isDataLoaded={dataLoaded}
            />
          </Stack.Item>
        </Stack>
      </Stack>
    </Panel>
  )
}
