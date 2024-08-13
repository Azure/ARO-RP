import * as React from "react"
import { useState, useEffect } from "react"
import { Stack, StackItem, IconButton, IIconStyles, SelectionMode } from "@fluentui/react"
import { Link } from "@fluentui/react/lib/Link"
import { IColumn } from "@fluentui/react/lib/DetailsList"
import { ShimmeredDetailsList } from "@fluentui/react/lib/ShimmeredDetailsList"
import { IClusterOperator } from "./ClusterOperatorsWrapper"
import { ClusterOperatorsComponent } from "./ClusterOperator"
import { _copyAndSort } from "../Utilities"

export declare interface IClusterOperatorList {
  name: string
  available: string
}

interface ClusterOperatorComponentProps {
  clusterOperators: IClusterOperator[]
  clusterName: string
}

export interface IClusterOperatorListState {
  clusterOperators: IClusterOperator[]
  clusterName: string
}

export class ClusterOperatorListComponent extends React.Component<
  ClusterOperatorComponentProps,
  IClusterOperatorListState
> {
  constructor(props: ClusterOperatorComponentProps) {
    super(props)

    this.state = {
      clusterOperators: this.props.clusterOperators,
      clusterName: this.props.clusterName,
    }
  }

  public render() {
    return (
      <ClusterOperatorListHelperComponent
        clusterOperators={this.state.clusterOperators}
        clusterName={this.state.clusterName}
      />
    )
  }
}

export function ClusterOperatorListHelperComponent(props: {
  clusterOperators: IClusterOperator[]
  clusterName: string
}) {
  const [columns, setColumns] = useState<IColumn[]>([
    {
      key: "clusterOperatorName",
      name: "Name",
      fieldName: "name",
      minWidth: 150,
      maxWidth: 350,
      isResizable: true,
      isPadded: true,
      showSortIconWhenUnsorted: true,
      isSortedDescending: false,
      isSorted: true,
      onRender: (item: IClusterOperatorList) => (
        <Link onClick={() => _onClusterOperatorInfoLinkClick(item.name)}>{item.name}</Link>
      ),
    },
    {
      key: "clusterOperatorAvailable",
      name: "Available",
      fieldName: "available",
      minWidth: 100,
      maxWidth: 100,
      isPadded: true,
      isResizable: true,
      isSortedDescending: false,
      isSorted: true,
      showSortIconWhenUnsorted: true,
    },
    {
      key: "clusterOperatorProgressing",
      name: "Progressing",
      fieldName: "progressing",
      minWidth: 100,
      maxWidth: 100,
      isPadded: true,
      isResizable: true,
      isSortedDescending: false,
      isSorted: true,
      showSortIconWhenUnsorted: true,
    },
    {
      key: "clusterOperatorDegraded",
      name: "Degraded",
      fieldName: "degraded",
      minWidth: 100,
      maxWidth: 100,
      isPadded: true,
      isResizable: true,
      isSortedDescending: false,
      isSorted: true,
      showSortIconWhenUnsorted: true,
    },
  ])

  const [clusterOperatorList, setClusterOperatorList] = useState<IClusterOperatorList[]>([])
  const [clusterOperatorDetailsVisible, setClusterOperatorDetailsVisible] = useState<boolean>(false)
  const [currentClusterOperator, setCurrentclusterOperator] = useState<string>("")
  const [shimmerVisibility, SetShimmerVisibility] = useState<boolean>(true)

  useEffect(() => {
    setClusterOperatorList(createClusterOperatorList(props.clusterOperators))
    const newColumns: IColumn[] = columns.slice()
    newColumns.forEach((col) => {
      col.onColumnClick = _onColumnClick
    })
    setColumns(newColumns)

    if (clusterOperatorList.length > 0) {
      SetShimmerVisibility(false)
    }
  }, [props.clusterOperators, clusterOperatorList])

  function _onClusterOperatorInfoLinkClick(operator: string) {
    setClusterOperatorDetailsVisible(!clusterOperatorDetailsVisible)
    setCurrentclusterOperator(operator)
  }

  function _onColumnClick(event: React.MouseEvent<HTMLElement>, column: IColumn): void {
    let clusterOperatorLocal: IClusterOperatorList[] = clusterOperatorList

    let isSortedDescending = column.isSortedDescending
    if (column.isSorted) {
      isSortedDescending = !isSortedDescending
    }

    // Sort the items.
    clusterOperatorLocal = _copyAndSort(clusterOperatorLocal, column.fieldName!, isSortedDescending)
    setClusterOperatorList(clusterOperatorLocal)

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

  function createClusterOperatorList(operators: IClusterOperator[]): IClusterOperatorList[] {
    return operators.map((operator) => {
      return {
        name: operator.name,
        available: operator.available,
        progressing: operator.progressing,
        degraded: operator.degraded,
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
  function _onClickBackToClusterOperatorList() {
    setClusterOperatorDetailsVisible(false)
  }

  return (
    <Stack>
      <StackItem>
        {clusterOperatorDetailsVisible ? (
          <Stack>
            <Stack.Item>
              <IconButton
                styles={backIconStyles}
                onClick={_onClickBackToClusterOperatorList}
                iconProps={backIconProp}
              />
            </Stack.Item>
            <ClusterOperatorsComponent
              clusterOperators={props.clusterOperators}
              clusterName={props.clusterName}
              clusterOperatorName={currentClusterOperator}
            />
          </Stack>
        ) : (
          <div>
            <ShimmeredDetailsList
              setKey="clusterOperatorList"
              compact={true}
              items={clusterOperatorList}
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
