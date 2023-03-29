import { IColumn, IconButton, IIconStyles, Link, SelectionMode, ShimmeredDetailsList, Stack, StackItem } from '@fluentui/react';
import * as React from 'react';
import { Component, useEffect, useState } from 'react';
import { ClusterNetworksComponent } from './ClusterNetworks';
import { IClusterNetwork } from './NetworkWrapper';


export declare interface IClusterNetworksList {
    name?: string;
    networkcidr: string;
    servicenetworkcidr: string;
}
  
interface ClusterNetworksListComponentProps {
    clusterNetworks: any
    clusterName: string
}
  
export interface IClusterNetworksListState {
    clusterNetworks: IClusterNetwork[]
    clusterName: string
}

export class ClusterNetworksListComponent extends Component<ClusterNetworksListComponentProps, IClusterNetworksListState> {

    constructor(props: ClusterNetworksListComponentProps) {
        super(props)

        this.state = {
            clusterNetworks: this.props.clusterNetworks,
            clusterName: this.props.clusterName,
        }
    }

    public render() {
        return (
            <ClusterNetworksListHelperComponent clusterNetworks={this.state.clusterNetworks} clusterName={this.state.clusterName}/>
          )
    }
}

export function ClusterNetworksListHelperComponent(props: {
    clusterNetworks: any,
    clusterName: string
}) {
    const [columns, setColumns] = useState<IColumn[]>([
        {
            key: "clusterNetworkName",
            name: "Name",
            fieldName: "name",
            minWidth: 150,
            maxWidth: 350,
            isResizable: true,
            isSorted: true,
            isSortedDescending: false,
            showSortIconWhenUnsorted: true,
            onRender: (item: IClusterNetworksList) => (
            <Link onClick={() => _onClusterNetworkInfoLinkClick(item.name!)}>{item.name}</Link>
            ),
        },
        {
            key: "clusterNetworkNetworkCidr",
            name: "Network CIDR",
            fieldName: "networkcidr",
            minWidth: 60,
            maxWidth: 120,
            isResizable: true,
            isSorted: true,
            isSortedDescending: false,
            showSortIconWhenUnsorted: true,
        },
        {
            key: "clusterNetworkServiceNetworkCidr",
            name: "Service CIDR",
            fieldName: "servicenetworkcidr",
            minWidth: 60,
            maxWidth: 80,
            isResizable: true,
            isSorted: true,
            isSortedDescending: false,
            showSortIconWhenUnsorted: true,
        }
    ])

    const [clusterNetworksList, setClusterNetworksList] = useState<IClusterNetworksList[]>([])
    const [clusterNetworksDetailsVisible, setClusterNetworksDetailsVisible] = useState<boolean>(false)
    const [currentClusterNetwork, setCurrentClusterNetwork] = useState<string>("")
    const [shimmerVisibility, SetShimmerVisibility] = useState<boolean>(true)

    useEffect(() => {
        setClusterNetworksList(createClusterNetworksList(props.clusterNetworks))
    }, [props.clusterNetworks] )

    useEffect(() => {
        const newColumns: IColumn[] = columns.slice();
        newColumns.forEach(col => {
        col.onColumnClick = _onColumnClick
        })
        setColumns(newColumns)

        if (clusterNetworksList.length > 0) {
        SetShimmerVisibility(false)
        }
        
    }, [clusterNetworksList])

    function _onClusterNetworkInfoLinkClick(clusterNetwork: string) {
        setClusterNetworksDetailsVisible(!clusterNetworksDetailsVisible)
        setCurrentClusterNetwork(clusterNetwork)
    }

    function _copyAndSort<T>(items: T[], columnKey: string, isSortedDescending?: boolean): T[] {
        const key = columnKey as keyof T;
        return items.slice(0).sort((a: T, b: T) => ((isSortedDescending ? a[key] < b[key] : a[key] > b[key]) ? 1 : -1));
    }

    function _onColumnClick(event: React.MouseEvent<HTMLElement>, column: IColumn): void {
        let clusterNetworkLocal: IClusterNetworksList[] = clusterNetworksList;
        
        let isSortedDescending = column.isSortedDescending;
        if (column.isSorted) {
            isSortedDescending = !isSortedDescending;
        }

        // Sort the items.
        clusterNetworkLocal = _copyAndSort(clusterNetworkLocal, column.fieldName!, isSortedDescending);
        setClusterNetworksList(clusterNetworkLocal)

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

    function createClusterNetworksList(clusterNetworks: IClusterNetwork[]): IClusterNetworksList[] {
        return clusterNetworks.map(clusterNetwork => {
            return {name: clusterNetwork.name, networkcidr: clusterNetwork.networkcidr, servicenetworkcidr: clusterNetwork.servicenetworkcidr}
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

    const backIconProp = {iconName: "back"}
      function _onClickBackToClusterNetworkList() {
        setClusterNetworksDetailsVisible(false)
    }

    return (
        <Stack>
          <StackItem>
            {
              clusterNetworksDetailsVisible
              ?
              <Stack>
                <Stack.Item>
                  <IconButton styles={backIconStyles} onClick={_onClickBackToClusterNetworkList} iconProps={backIconProp} />
                </Stack.Item>
                <ClusterNetworksComponent clusterNetworks={props.clusterNetworks} clusterName={props.clusterName} clusterNetworkName={currentClusterNetwork}/>
              </Stack>
              :
              <div>
              <ShimmeredDetailsList
                setKey="none"
                items={clusterNetworksList}
                columns={columns}
                selectionMode={SelectionMode.none}
                enableShimmer={shimmerVisibility}
                ariaLabelForShimmer="Content is being fetched"
                ariaLabelForGrid="Item details"
              />
              </div>
            }
          </StackItem>
        </Stack>
      )
}