import * as React from "react"
import { useState, useEffect } from "react"
import { Stack, StackItem, IconButton, IIconStyles, SelectionMode } from "@fluentui/react"
import { Link } from "@fluentui/react/lib/Link"
import { IColumn } from "@fluentui/react/lib/DetailsList"
import { ShimmeredDetailsList } from "@fluentui/react/lib/ShimmeredDetailsList"
import { INode } from "./NodesWrapper"
import { NodesComponent } from "./Nodes"
import { _copyAndSort } from "../Utilities"

export declare interface INodeList {
  name: string
  status: string
  schedulable: string
  instanceType?: string
}

interface NodeListComponentProps {
  nodes: any
  clusterName: string
}

export interface INodeListState {
  nodes: INode[]
  clusterName: string
}

export class NodesListComponent extends React.Component<NodeListComponentProps, INodeListState> {
  constructor(props: NodeListComponentProps) {
    super(props)

    this.state = {
      nodes: this.props.nodes,
      clusterName: this.props.clusterName,
    }
  }

  public render() {
    return <NodeListHelperComponent nodes={this.state.nodes} clusterName={this.state.clusterName} />
  }
}

export function NodeListHelperComponent(props: { nodes: any; clusterName: string }) {
  const [columns, setColumns] = useState<IColumn[]>([
    {
      key: "nodeName",
      name: "Name",
      fieldName: "name",
      minWidth: 150,
      maxWidth: 350,
      isResizable: true,
      isPadded: true,
      showSortIconWhenUnsorted: true,
      isSortedDescending: false,
      isSorted: true,
      onRender: (item: INodeList) => (
        <Link onClick={() => _onNodeInfoLinkClick(item.name)}>{item.name}</Link>
      ),
    },
    {
      key: "nodeStatus",
      name: "Status",
      fieldName: "status",
      minWidth: 50,
      maxWidth: 50,
      isPadded: true,
      isResizable: true,
      isSortedDescending: false,
      isSorted: true,
      showSortIconWhenUnsorted: true,
    },
    {
      key: "nodeSchedulable",
      name: "Schedulable",
      fieldName: "schedulable",
      minWidth: 70,
      maxWidth: 70,
      isPadded: true,
      isResizable: true,
      isSortedDescending: false,
      isSorted: true,
      showSortIconWhenUnsorted: true,
    },
    {
      key: "nodeInstanceType",
      name: "Instance Type",
      fieldName: "instanceType",
      minWidth: 80,
      maxWidth: 80,
      isPadded: true,
      isResizable: true,
      isSortedDescending: false,
      isSorted: true,
      showSortIconWhenUnsorted: true,
    },
  ])

  const [nodeList, setNodesList] = useState<INodeList[]>([])
  const [nodeDetailsVisible, setNodesDetailsVisible] = useState<boolean>(false)
  const [currentNode, setCurrentNode] = useState<string>("")
  const [shimmerVisibility, SetShimmerVisibility] = useState<boolean>(true)

  useEffect(() => {
    setNodesList(createNodeList(props.nodes))
  }, [props.nodes])

  useEffect(() => {
    const newColumns: IColumn[] = columns.slice()
    newColumns.forEach((col) => {
      col.onColumnClick = _onColumnClick
    })
    setColumns(newColumns)

    if (nodeList.length > 0) {
      SetShimmerVisibility(false)
    }
  }, [nodeList])

  function _onNodeInfoLinkClick(node: string) {
    setNodesDetailsVisible(!nodeDetailsVisible)
    setCurrentNode(node)
  }

  function _onColumnClick(event: React.MouseEvent<HTMLElement>, column: IColumn): void {
    let nodeLocal: INodeList[] = nodeList

    let isSortedDescending = column.isSortedDescending
    if (column.isSorted) {
      isSortedDescending = !isSortedDescending
    }

    // Sort the items.
    nodeLocal = _copyAndSort(nodeLocal, column.fieldName!, isSortedDescending)
    setNodesList(nodeLocal)

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
    setColumns(newColumns)
  }

  function createNodeList(nodes: INode[]): INodeList[] {
    return nodes.map((node) => {
      let schedulable: string = "True"
      let instanceType: string = node.labels?.get("node.kubernetes.io/instance-type")!

      if (node.conditions![3].status === "True") {
        node.taints?.forEach((taint) => {
          schedulable = taint.key === "node.kubernetes.io/unschedulable" ? "False" : "True"
        })

        return {
          name: node.name,
          status: "Ready",
          schedulable: schedulable!,
          instanceType: instanceType,
        }
      } else {
        schedulable = "--"
        return {
          name: node.name,
          status: "Not Ready",
          schedulable: schedulable!,
          instanceType: instanceType,
        }
      }
    })
  }

  const backIconStyles: Partial<IIconStyles> = {
    root: {
      height: "100%",
      width: 40,
      paddingTop: 5,
      paddingBottam: 15,
      svg: {
        fill: "#e3222f",
      },
    },
  }

  const backIconProp = { iconName: "back" }
  function _onClickBackToNodeList() {
    setNodesDetailsVisible(false)
  }

  return (
    <Stack>
      <StackItem>
        {nodeDetailsVisible ? (
          <Stack>
            <Stack.Item>
              <IconButton
                styles={backIconStyles}
                onClick={_onClickBackToNodeList}
                iconProps={backIconProp}
              />
            </Stack.Item>
            <NodesComponent
              nodes={props.nodes}
              clusterName={props.clusterName}
              nodeName={currentNode}
            />
          </Stack>
        ) : (
          <div>
            <ShimmeredDetailsList
              setKey="nodeList"
              items={nodeList}
              columns={columns}
              selectionMode={SelectionMode.none}
              enableShimmer={shimmerVisibility}
              ariaLabelForShimmer="Content is being fetched"
              ariaLabelForGrid="Item details"
            />
          </div>
        )}
      </StackItem>
    </Stack>
  )
}
