import React, { useState, useEffect, useRef, MutableRefObject, Component } from "react"
import {
  Stack,
  MessageBarType,
  MessageBar,
  CommandBar,
  ICommandBarItemProps,
  Separator,
  Text,
  IMessageBarStyles,
  mergeStyleSets,
  TextField,
  Link,
} from "@fluentui/react"
import {
  DetailsList,
  DetailsListLayoutMode,
  SelectionMode,
  IColumn,
  IDetailsListStyles,
} from "@fluentui/react/lib/DetailsList"
import { fetchClusters } from "./Request"
import { ToolIcons } from "./ToolIcons"
import { ICluster, headerStyles } from "./App"
import { useHref, useLinkClickHandler } from "react-router"

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
        onRender: (item: ICluster) => {
          const href = useHref(item.resourceId)
          const onClick = useLinkClickHandler(item.resourceId)
          return (
            <Link href={href} onClick={(ev) => onClick(ev as React.MouseEvent<HTMLAnchorElement>)}>
              {item.name}
            </Link>
          )
        },
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
            <ToolIcons
              resourceId={item.resourceId}
              csrfToken={props.csrfToken}
              version={Number(item.version)}
              sshBox={props.sshModalRef}
            />
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
        <Text id="ClusterCount" className={classNames.itemsCount}>
          Showing {items.length} items
        </Text>
        <DetailsList
          items={items}
          columns={columns}
          selectionMode={SelectionMode.none}
          getKey={this._getKey}
          setKey="none"
          layoutMode={DetailsListLayoutMode.fixedColumns}
          isHeaderVisible={true}
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
        ? this.props.items.filter(
            (i) => i.resourceId.toLowerCase().indexOf(text.trim().toLowerCase()) != -1
          )
        : this.props.items,
    })
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
  csrfTokenAvailable: string
}) {
  const [data, setData] = useState<any>([])
  const [error, setError] = useState<Response | null>(null)
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
    const onData = async (result: Response) => {
      if (result.status === 200) {
        updateData(await result.json())
      } else {
        setError(result)
      }
      setFetching("DONE")
    }

    if (fetching === "" && props.csrfTokenAvailable === "DONE") {
      setFetching("FETCHING")
      fetchClusters().then(onData)
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

      <ClusterListComponent
        items={data}
        ref={state} // why do we need ref here?
        sshModalRef={props.sshBox}
        csrfToken={props.csrfToken}
      />
    </Stack>
  )
}
