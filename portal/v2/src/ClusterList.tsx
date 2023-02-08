import React, { useState, useEffect, useRef, MutableRefObject, Component } from "react"
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
  Layer,
  Popup,
  DefaultButton,
  FocusTrapZone,
} from "@fluentui/react"
import {
  DetailsList,
  DetailsListLayoutMode,
  SelectionMode,
  IColumn,
  IDetailsListStyles,
} from "@fluentui/react/lib/DetailsList"
import { useBoolean } from "@fluentui/react-hooks"
import { FetchClusters } from "./Request"
import { KubeconfigButton } from "./Kubeconfig"
import { AxiosResponse } from "axios"
import { ICluster, headerStyles } from "./App"

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

const popupStyles = mergeStyleSets({
  root: {
    background: 'rgba(0, 0, 0, 0.2)',
    bottom: '0',
    left: '0',
    position: 'fixed',
    right: '0',
    top: '0',
  },
  content: {
    background: 'white',
    left: '50%',
    maxWidth: '400px',
    padding: '0 2em 2em',
    position: 'absolute',
    top: '50%',
    transform: 'translate(-50%, -50%)',
  },
});

const PopupModal = (props: {title: string, text: string, hidePopup: any}) => {
  return (
    <>
        <Layer>
          <Popup
            className={popupStyles.root}
            role="dialog"
            aria-modal="true"
            onDismiss={props.hidePopup}
            enableAriaHiddenSiblings={true}
          >
            <FocusTrapZone>
              <div role="document" className={popupStyles.content}>
                <h2>{props.title}</h2>
                <p>
                  {props.text}
                </p>
                <DefaultButton onClick={() => {
                  // this is to change the URL in the address bar
                  window.history.replaceState({}, "", "/v2")
                  props.hidePopup()
                }}>
                  Close
                </DefaultButton>
              </div>
            </FocusTrapZone>
          </Popup>
        </Layer>
    </>
  );
};

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

/* eslint-disable */

interface ClusterListComponentProps {
  items: ICluster[]
  sshModalRef: MutableRefObject<any>
  setCurrentCluster: (item: ICluster) => void
  csrfToken: MutableRefObject<string>
}

/* eslint-enable */
class ClusterListComponent extends Component<ClusterListComponentProps, IClusterListState> {
  private _sshModal: MutableRefObject<any>

  constructor(props: ClusterListComponentProps) {
    super(props)

    this._sshModal = props.sshModalRef

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
        onRender: () => (
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
          <Link onClick={() => this._onClusterInfoLinkClick(item)}>{item.name}</Link>
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
        minWidth: 50,
        flexGrow: 3,
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
        fieldName: "createdAt",
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
        fieldName: "provisioningState",
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
            {item.provisioningState}
            {item.failedProvisioningState && " - " + item.failedProvisioningState}
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
            <TooltipHost content={`Copy Resource ID`}>
              <IconButton
                iconProps={{ iconName: "Copy" }}
                aria-label="Copy Resource ID"
                onClick={() => this._onCopyResourceID(item)}
              />
            </TooltipHost>
            <TooltipHost content={`Prometheus`}>
              <IconButton
                iconProps={{ iconName: "BIDashboard" }}
                aria-label="Prometheus"
                href={item.resourceId + (+item.version >= 4.11 ? `/prometheus` : `/prometheus/graph`)}
              />
            </TooltipHost>
            <TooltipHost content={`SSH`}>
              <IconButton
                iconProps={{ iconName: "CommandPrompt" }}
                aria-label="SSH"
                onClick={() => this._onSSHClick(item)}
              />
            </TooltipHost>
            <KubeconfigButton resourceId={item.resourceId} csrfToken={props.csrfToken} />
            {/* <TooltipHost content={`Geneva`}>
              <IconButton
                iconProps={{iconName: "Health"}}
                aria-label="Geneva"
                href={item.resourceId + `/geneva`}
              />
            </TooltipHost>
            <TooltipHost content={`Feature Flags`}>
              <IconButton
                iconProps={{iconName: "IconSetsFlag"}}
                aria-label="featureFlags"
                href={item.resourceId + `/feature-flags`}
              />
            </TooltipHost> */}
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
          <TextField placeholder="Filter on resource ID" onChange={this._onChangeText} />
        </div>
        <Text id="ClusterCount" className={classNames.itemsCount}>Showing {items.length} items</Text>
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

  private _getKey(item: any): string {
    return item.key
  }

  private _onChangeText = (
    ev: React.FormEvent<HTMLInputElement | HTMLTextAreaElement>,
    text?: string
  ): void => {
    this.setState({
      items: text
      ? this.props.items.filter((i) => i.resourceId.toLowerCase().indexOf(text.trim().toLowerCase()) != -1)
        : this.props.items,
    })
  }

  private _onSSHClick(item: any): void {
    const modal = this._sshModal
    if (modal && modal.current) {
      modal.current.LoadSSH(item.resourceId)
    }
  }

  private _onCopyResourceID(item: any): void {
    navigator.clipboard.writeText(item.resourceId)
  }

  private _onClusterInfoLinkClick(item: ICluster): void {
    this.props.setCurrentCluster(item)
  }

  private _onItemInvoked(item: any): void {
    alert(`Item invoked: ${item.resourceId}`)
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
  setCurrentCluster: any
  csrfTokenAvailable: string
  params: any
}) {
  const [data, setData] = useState<any>([])
  const [error, setError] = useState<AxiosResponse | null>(null)
  const [isPopupVisible, { setTrue: showPopup, setFalse: hidePopup }] = useBoolean(false);
  const state = useRef<ClusterListComponent>(null)
  const [fetching, setFetching] = useState("")

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

    if (fetching === "" && props.csrfTokenAvailable === "DONE") {
      setFetching("FETCHING")
      FetchClusters().then(onData)
    }

    if (props.params) {
      const resourceID: string = props.params["resourceid"]
      const clusterList = data as ICluster[]
      const currentCluster = clusterList.find((item): item is ICluster => resourceID === item.resourceId)

      if (fetching === "DONE" && !currentCluster) {
        showPopup()
        return
      }

      props.setCurrentCluster(currentCluster)
    }

  }, [data, fetching, setFetching, props.csrfTokenAvailable])

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
      <span className={headerStyles.titleText}>Clusters</span>
      <span className={headerStyles.subtitleText}>Azure Red Hat OpenShift</span>
      <CommandBar
        items={_items}
        ariaLabel="Use left and right arrow keys to navigate between commands"
        className={classNames.controlButtonContainer}
        styles={controlStyles}
      />
      <Separator styles={separatorStyle} />
      
      {error && errorBar()}

      {isPopupVisible && PopupModal({title: "Resource Not Found", text: "No resource found due to Invalid/Non-existent resource ID in the URL.", hidePopup: hidePopup})}

      <ClusterListComponent
        items={data}
        ref={state} // why do we need ref here?
        sshModalRef={props.sshBox}
        setCurrentCluster={props.setCurrentCluster}
        csrfToken={props.csrfToken}
      />
    </Stack>
  )
}
