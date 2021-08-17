import React, {useState, useEffect, useRef, MutableRefObject, Component} from "react"
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
} from "@fluentui/react"
import {
  DetailsList,
  DetailsListLayoutMode,
  SelectionMode,
  IColumn,
  IDetailsListStyles,
} from "@fluentui/react/lib/DetailsList"

import {FetchClusters} from "./Request"
import {KubeconfigButton} from "./Kubeconfig"
import {AxiosResponse} from "axios"

interface ICluster {
  key: string
  name: string
}

const errorBarStyles: Partial<IMessageBarStyles> = {root: {marginBottom: 15}}

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
  csrfToken: MutableRefObject<string>
}

class ClusterListControl extends Component<ClusterListControlProps, IClusterListState> {
  private _sshModal: MutableRefObject<any>

  constructor(props: ClusterListControlProps) {
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
        minWidth: 210,
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
        key: "icons",
        name: "",
        fieldName: "icons",
        minWidth: 92,
        isRowHeader: false,
        data: "string",
        isPadded: true,
        onRender: (item: ICluster) => (
          <Stack horizontal verticalAlign="center" className={classNames.iconContainer}>
            <TooltipHost content={`Prometheus`}>
              <IconButton
                iconProps={{iconName: "BarChart4"}}
                aria-label="Prometheus"
                href={item.name + `/prometheus`}
              />
            </TooltipHost>
            <TooltipHost content={`SSH`}>
              <IconButton
                iconProps={{iconName: "CommandPrompt"}}
                aria-label="SSH"
                onClick={(_) => this._onSSHClick(item)}
              />
            </TooltipHost>
            <KubeconfigButton clusterID={item.name} csrfToken={props.csrfToken} />
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
    const {columns, items} = this.state

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
          layoutMode={DetailsListLayoutMode.justified}
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

  private _onItemInvoked(item: any): void {
    alert(`Item invoked: ${item.name}`)
  }

  private _onColumnClick = (ev: React.MouseEvent<HTMLElement>, column: IColumn): void => {
    const {columns, items} = this.state
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
      state.current.setState({items: newData})
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
      iconProps: {iconName: "Refresh"},
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
        csrfToken={props.csrfToken}
      />
    </Stack>
  )
}
