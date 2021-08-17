import React, { useState, useEffect, useRef, MutableRefObject, Component, SyntheticEvent } from "react"
import {
  Stack,
  IconButton,
  MessageBarType,
  MessageBar,
  CommandBar,
  ICommandBarItemProps,
  Separator,
  Text,
  IMessageBarStyles,
  mergeStyleSets,
  TooltipHost,
  TextField,
  Link,
  ShimmeredDetailsList,
  registerIcons,
} from "@fluentui/react"
import {
  DetailsList,
  DetailsListLayoutMode,
  SelectionMode,
  IColumn,
  IDetailsListStyles,
} from "@fluentui/react/lib/DetailsList"

import { FetchClusters, FetchClusterInfo } from "./Request"
import { KubeconfigButton } from "./Kubeconfig"
import { AxiosResponse } from "axios"

var currentName: string
var currentSubscription: string
var currentResourceGroup: string

registerIcons({
  icons: {
    'openshift-svg': (
      <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 64 64">
        <g fill="#0078d4">
          <path d="M17.424 27.158L7.8 30.664c.123 1.545.4 3.07.764 4.566l9.15-3.333c-.297-1.547-.403-3.142-.28-4.74M60 16.504c-.672-1.386-1.45-2.726-2.35-3.988l-9.632 3.506c1.12 1.147 2.06 2.435 2.83 3.813z" />
          <path d="M38.802 13.776c2.004.935 3.74 2.21 5.204 3.707l9.633-3.506a27.38 27.38 0 0 0-10.756-8.95c-13.77-6.42-30.198-.442-36.62 13.326a27.38 27.38 0 0 0-2.488 13.771l9.634-3.505c.16-2.087.67-4.18 1.603-6.184 4.173-8.947 14.844-12.83 23.79-8.658" />
        </g>
        <path d="M9.153 35.01L0 38.342c.84 3.337 2.3 6.508 4.304 9.33l9.612-3.5a17.99 17.99 0 0 1-4.763-9.164" fill="#0078d4" />
        <path d="M49.074 31.38a17.64 17.64 0 0 1-1.616 6.186c-4.173 8.947-14.843 12.83-23.79 8.657a17.71 17.71 0 0 1-5.215-3.7l-9.612 3.5c2.662 3.744 6.293 6.874 10.748 8.953 13.77 6.42 30.196.44 36.618-13.328a27.28 27.28 0 0 0 2.479-13.765l-9.61 3.498z" fill="#0078d4" />
        <path d="M51.445 19.618l-9.153 3.332c1.7 3.046 2.503 6.553 2.24 10.08l9.612-3.497c-.275-3.45-1.195-6.817-2.7-9.915" fill="#0078d4" />
      </svg>
    )
  },
})

interface ICluster {
  key: string
  name: string
  subscription: string
  resourceGroup: string
  id: string
  version: string
  createdDate: string
  provisionedBy: string
  lastModified: string
  state: string
  failed: string
  consoleLink: string
}

const errorBarStyles: Partial<IMessageBarStyles> = { root: { marginBottom: 15 } }

const classNames = mergeStyleSets({
  controlWrapper: {
    display: "flex",
    flexWrap: "wrap",
  },
  fullWidth: {
    width: "100%",
  },
  fileIconImg: {
    verticalAlign: "middle",
    maxHeight: "20px",
    maxWidth: "20px",
  },
  headerIcon: {
    height: 18,
    paddingTop: 1,
  },
  iconContainer: {
    margin: "-11px 0px",
    height: 42,
  },
  controlButtonContainer: {
    paddingLeft: 0,
  },
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
  itemsCount: {
    padding: "10px 0px",
  },
})

const controlStyles = {
  root: {
    paddingLeft: 0,
  },
}

const separatorStyle = {
  root: {
    fontSize: 0,
    marginBottom: 20,
    padding: 0,
  },
}

interface IClusterListState {
  columns: IColumn[]
  items: ICluster[]
  modalOpen: boolean
}

const clusterListDetailStyles: Partial<IDetailsListStyles> = {
  headerWrapper: {
    marginTop: "-16px",
  },
}

interface ClusterListControlProps {
  items: ICluster[]
  sshModalRef: MutableRefObject<any>
  clusterDetailPanelRef: MutableRefObject<any>
  csrfToken: MutableRefObject<string>
}
function handleClickOnLink(ev: React.MouseEvent<unknown>) {
  // {currentName = item.name}
  // {currentSubscription = item.subscription}
  // {currentResourceGroup = item.resourceGroup}
  FetchClusterInfo(currentSubscription, currentResourceGroup, currentName).then(
    function (result) {
      console.log(result?.data)
    })
}

class ClusterListControl extends Component<ClusterListControlProps, IClusterListState> {
  private _sshModal: MutableRefObject<any>
  private _clusterDetailPanel: MutableRefObject<any>

  constructor(props: ClusterListControlProps) {
    super(props)

    this._sshModal = props.sshModalRef
    this._clusterDetailPanel = props.clusterDetailPanelRef

    const columns: IColumn[] = [
      {
        key: "icon",
        name: "",
        fieldName: "",
        minWidth: 24,
        isRowHeader: false,
        data: "string",
        isPadded: false,
        maxWidth: 24,
        onRender: (item: ICluster) => (
          <Stack horizontal verticalAlign="center" className={classNames.iconContainer}>
            <img src="/favicon.ico" className={classNames.headerIcon} alt="" />
          </Stack>
        ),
      },
      {
        key: "name",
        name: "Name",
        fieldName: "name",
        minWidth: 100,
        flexGrow: 10,
        isRowHeader: true,
        isResizable: true,
        isSorted: true,
        isSortedDescending: false,
        sortAscendingAriaLabel: "Sorted A to Z",
        sortDescendingAriaLabel: "Sorted Z to A",
        onColumnClick: this._onColumnClick,
        data: "string",
        onRender: (item: ICluster) => (
          <Link onClick={(_) => this._onClusterDetailPanelClick(item)} >
            {item.name}
          </Link>
        ),
        isPadded: true,
      },
      {
        key: "subscription",
        name: "Subscription",
        fieldName: "subscription",
        minWidth: 100,
        flexGrow: 10,
        isRowHeader: true,
        isResizable: true,
        isSorted: true,
        isSortedDescending: false,
        sortAscendingAriaLabel: "Sorted A to Z",
        sortDescendingAriaLabel: "Sorted Z to A",
        onColumnClick: this._onColumnClick,
        data: "string",
        isPadded: true,
      },
      {
        key: "version",
        name: "Version",
        fieldName: "version",
        minWidth: 100,
        flexGrow: 5,
        isRowHeader: true,
        isResizable: true,
        isSorted: true,
        isSortedDescending: false,
        sortAscendingAriaLabel: "Sorted A to Z",
        sortDescendingAriaLabel: "Sorted Z to A",
        onColumnClick: this._onColumnClick,
        data: "string",
        isPadded: true,
      },
      {
        key: "latestModified",
        name: "Last Modified",
        fieldName: "lastModified",
        minWidth: 100,
        flexGrow: 5,
        isRowHeader: true,
        isResizable: true,
        isSorted: true,
        isSortedDescending: false,
        sortAscendingAriaLabel: "Sorted A to Z",
        sortDescendingAriaLabel: "Sorted Z to A",
        onColumnClick: this._onColumnClick,
        data: "string",
        isPadded: true,
      },
      {
        key: "createdDate",
        name: "Creation Date",
        fieldName: "createdDate",
        minWidth: 100,
        flexGrow: 5,
        isRowHeader: true,
        isResizable: true,
        isSorted: true,
        isSortedDescending: false,
        sortAscendingAriaLabel: "Sorted A to Z",
        sortDescendingAriaLabel: "Sorted Z to A",
        onColumnClick: this._onColumnClick,
        data: "string",
        isPadded: true,
      },
      {
        key: "provisionedBy",
        name: "Provisioned By",
        fieldName: "provisionedBy",
        minWidth: 100,
        flexGrow: 5,
        isRowHeader: true,
        isResizable: true,
        isSorted: true,
        isSortedDescending: false,
        sortAscendingAriaLabel: "Sorted A to Z",
        sortDescendingAriaLabel: "Sorted Z to A",
        onColumnClick: this._onColumnClick,
        data: "string",
        isPadded: true,
      },
      {
        key: "state",
        name: "State",
        fieldName: "state",
        minWidth: 100,
        flexGrow: 5,
        isRowHeader: true,
        isResizable: true,
        isSorted: true,
        isSortedDescending: false,
        sortAscendingAriaLabel: "Sorted A to Z",
        sortDescendingAriaLabel: "Sorted Z to A",
        onColumnClick: this._onColumnClick,
        onRender: (item: ICluster) => (
          <Text>
            {item.state}{item.failed && " - " + item.failed}
          </Text>
        ),
        data: "string",
        isPadded: true,
      },
      {
        key: "icons",
        name: "Actions",
        fieldName: "icons",
        minWidth: 92,
        flexGrow: 5,
        isRowHeader: false,
        data: "string",
        isPadded: true,
        onRender: (item: ICluster) => (
          <Stack horizontal verticalAlign="center" className={classNames.iconContainer}>
            <TooltipHost content={`Prometheus`}>
              <IconButton
                iconProps={{ iconName: "BIDashboard" }}
                aria-label="Prometheus"
                href={item.name + `/prometheus`}
              />
            </TooltipHost>
            <TooltipHost content={`OpenShift Console`}>
              <IconButton
                iconProps={{ iconName: "openshift-svg" }}
                aria-label="Console"
                href={item.consoleLink}
              />
            </TooltipHost>
            <TooltipHost content={`SSH`}>
              <IconButton
                iconProps={{ iconName: "CommandPrompt" }}
                aria-label="SSH"
                onClick={(_) => this._onSSHClick(item)}
              />
            </TooltipHost>
            <KubeconfigButton clusterID={item.name} csrfToken={props.csrfToken} />
            <TooltipHost content={`Geneva`}>
              <IconButton
                iconProps={{ iconName: "Health" }}
                aria-label="Geneva"
                href={item.name + `/geneva`}
              />
            </TooltipHost>
            <TooltipHost content={`Upgrade`}>
              <IconButton
                iconProps={{ iconName: "Up" }}
                aria-label="upgrade"
                href={item.name + `/upgrade`}
              />
            </TooltipHost>
            <TooltipHost content={`Feature Flags`}>
              <IconButton
                iconProps={{ iconName: "IconSetsFlag" }}
                aria-label="featureFlags"
                href={item.name + `/feature-flags`}
              />
            </TooltipHost>
          </Stack>
        ),
      },
    ]

    this.state = {
      items: this.props.items,
      columns: columns,
      modalOpen: false,
    }
  }

  public render() {
    const { columns, items } = this.state

    return (
      <Stack>
        <div className={classNames.controlWrapper}>
          <TextField placeholder="Filter on any field" onChange={this._onChangeText} />
        </div>
        <Text className={classNames.itemsCount}>Showing {items.length} items</Text>
        <DetailsList
          items={items}
          columns={columns}
          selectionMode={SelectionMode.none}
          getKey={this._getKey}
          setKey="none"
          layoutMode={DetailsListLayoutMode.fixedColumns}
          isHeaderVisible={true}
          onItemInvoked={this._onItemInvoked}
          styles={clusterListDetailStyles}
        />
      </Stack>
    )
  }

  private _getKey(item: any, index?: number): string {
    return item.key
  }

  private _onChangeText = (
    ev: React.FormEvent<HTMLInputElement | HTMLTextAreaElement>,
    text?: string
  ): void => {
    this.setState({
      items: text
        ? this.props.items.filter((i) => i.name.toLowerCase().indexOf(text) > -1)
        : this.props.items,
    })
  }

  private _onSSHClick(item: any): void {
    const modal = this._sshModal
    if (modal && modal.current) {
      modal.current.LoadSSH(item.name)
    }
  }

  private _onClusterDetailPanelClick(item: any): void {
    const panel = this._clusterDetailPanel
    if (panel && panel.current) {
      panel.current.LoadClusterDetailPanel(item)
    }
  }

  private _onItemInvoked(item: any): void {
    alert(`Item invoked: ${item.name}`)
  }

  private _onColumnClick = (ev: React.MouseEvent<HTMLElement>, column: IColumn): void => {
    const { columns, items } = this.state
    const newColumns: IColumn[] = columns.slice()
    const currColumn: IColumn = newColumns.filter((currCol) => column.key === currCol.key)[0]
    newColumns.forEach((newCol: IColumn) => {
      if (newCol === currColumn) {
        currColumn.isSortedDescending = !currColumn.isSortedDescending
        currColumn.isSorted = true
      } else {
        newCol.isSorted = false
        newCol.isSortedDescending = true
      }
    })
    this.setState({
      columns: newColumns,
      items: items,
    })
  }
}

export function ClusterList(props: {
  csrfToken: MutableRefObject<string>
  sshBox: MutableRefObject<any>
  clusterDetailPanel: MutableRefObject<any>
  loaded: string
}) {
  const [data, setData] = useState<any>([])
  const [error, setError] = useState<AxiosResponse | null>(null)
  const state = useRef<ClusterListControl>(null)
  const [fetching, setFetching] = useState("")

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

  // Helper function to refresh the actual state of the DetailList
  // see https://developer.microsoft.com/en-us/fluentui#/controls/web/detailslist#best-practices
  const updateData = (newData: any) => {
    setData(newData)
    if (state && state.current) {
      state.current.setState({ items: newData })
    }
  }

  useEffect(() => {
    const onData = (result: AxiosResponse | null) => {
      if (result?.status === 200) {
        updateData(result.data)
      } else {
        setError(result)
      }
      setFetching("DONE")
    }

    if (fetching === "" && props.loaded === "DONE") {
      setFetching("FETCHING")
      FetchClusters().then(onData)
    }
  }, [data, fetching, setFetching, props.loaded])

  const _items: ICommandBarItemProps[] = [
    {
      key: "refresh",
      text: "Refresh",
      iconProps: { iconName: "Refresh" },
      onClick: () => {
        updateData([])
        setFetching("")
      },
    },
  ]

  return (
    <Stack>
      <span className={classNames.titleText}>Clusters</span>
      <span className={classNames.subtitleText}>Azure Red Hat OpenShift</span>
      <CommandBar
        items={_items}
        ariaLabel="Use left and right arrow keys to navigate between commands"
        className={classNames.controlButtonContainer}
        styles={controlStyles}
      />
      <Separator styles={separatorStyle} />

      {error && errorBar()}
      <ClusterListControl
        items={data}
        ref={state}
        sshModalRef={props.sshBox}
        clusterDetailPanelRef={props.clusterDetailPanel}
        csrfToken={props.csrfToken}
      />
    </Stack>
  )
}
