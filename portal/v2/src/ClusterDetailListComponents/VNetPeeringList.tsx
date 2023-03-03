import { IColumn, IconButton, IIconStyles, Link, SelectionMode, ShimmeredDetailsList, Stack, StackItem } from '@fluentui/react';
import * as React from 'react';
import { useEffect, useState } from 'react';
import { IVNetPeering } from "./NetworkWrapper";
import { VNetPeeringsComponent } from './VNetPeerings';


export declare interface IVNetPeeringsList {
    name?: string;
    remotevnet: string;
    state: string;
}
  
interface VNetPeeringsListComponentProps {
    vNetPeerings: any
    clusterName: string
}
  
export interface IVNetPeeringsListState {
    vNetPeerings: IVNetPeering[]
    clusterName: string
}

export class VNetPeeringsListComponent extends React.Component<VNetPeeringsListComponentProps, IVNetPeeringsListState> {

    constructor(props: VNetPeeringsListComponentProps) {
        super(props)

        this.state = {
            vNetPeerings: this.props.vNetPeerings,
            clusterName: this.props.clusterName,
        }
    }

    public render() {
        return (
            <VNetPeeringsListHelperComponent vNetPeerings={this.state.vNetPeerings} clusterName={this.state.clusterName}/>
          )
    }
}

export function VNetPeeringsListHelperComponent(props: {
    vNetPeerings: any,
    clusterName: string
}) {
    const [columns, setColumns] = useState<IColumn[]>([
        {
            key: "vNetPeeringName",
            name: "Name",
            fieldName: "name",
            minWidth: 150,
            maxWidth: 350,
            isResizable: true,
            isSorted: true,
            isSortedDescending: false,
            showSortIconWhenUnsorted: true,
            onRender: (item: IVNetPeeringsList) => (
            <Link onClick={() => _onVNetPeeringInfoLinkClick(item.name!)}>{item.name}</Link>
            ),
        },
        {
            key: "vNetPeeringRemoteVNet",
            name: "Remote VNet",
            fieldName: "remotevnet",
            minWidth: 60,
            maxWidth: 120,
            isResizable: true,
            isSorted: true,
            isSortedDescending: false,
            showSortIconWhenUnsorted: true,
        },
        {
            key: "vNetPeeringState",
            name: "State",
            fieldName: "state",
            minWidth: 60,
            maxWidth: 80,
            isResizable: true,
            isSorted: true,
            isSortedDescending: false,
            showSortIconWhenUnsorted: true,
        }
    ])

    const [vNetPeeringsList, setVNetPeeringsList] = useState<IVNetPeeringsList[]>([])
    const [vNetPeeringsDetailsVisible, setVNetPeeringsDetailsVisible] = useState<boolean>(false)
    const [currentVNetPeering, setCurrentVNetPeering] = useState<string>("")
    const [shimmerVisibility, SetShimmerVisibility] = useState<boolean>(true)

    useEffect(() => {
        setVNetPeeringsList(createVNetPeeringsList(props.vNetPeerings))
    }, [props.vNetPeerings] )

    useEffect(() => {
        const newColumns: IColumn[] = columns.slice();
        newColumns.forEach(col => {
        col.onColumnClick = _onColumnClick
        })
        setColumns(newColumns)

        if (vNetPeeringsList.length > 0) {
        SetShimmerVisibility(false)
        }
        
    }, [vNetPeeringsList])

    function _onVNetPeeringInfoLinkClick(vNetPeering: string) {
        setVNetPeeringsDetailsVisible(!vNetPeeringsDetailsVisible)
        setCurrentVNetPeering(vNetPeering)
    }

    function _copyAndSort<T>(items: T[], columnKey: string, isSortedDescending?: boolean): T[] {
        const key = columnKey as keyof T;
        return items.slice(0).sort((a: T, b: T) => ((isSortedDescending ? a[key] < b[key] : a[key] > b[key]) ? 1 : -1));
    }

    function _onColumnClick(event: React.MouseEvent<HTMLElement>, column: IColumn): void {
        let vNetPeeringLocal: IVNetPeeringsList[] = vNetPeeringsList;
        
        let isSortedDescending = column.isSortedDescending;
        if (column.isSorted) {
            isSortedDescending = !isSortedDescending;
        }

        // Sort the items.
        vNetPeeringLocal = _copyAndSort(vNetPeeringLocal, column.fieldName!, isSortedDescending);
        setVNetPeeringsList(vNetPeeringLocal)

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

    function createVNetPeeringsList(vNetPeerings: IVNetPeering[]): IVNetPeeringsList[] {
        return vNetPeerings.map(vNetPeering => {
            return {name: vNetPeering.name, remotevnet: vNetPeering.remotevnet, state: vNetPeering.state}
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
      function _onClickBackToVNetPeeringList() {
        setVNetPeeringsDetailsVisible(false)
    }

    return (
        <Stack>
          <StackItem>
            {
              vNetPeeringsDetailsVisible
              ?
              <Stack>
                <Stack.Item>
                  <IconButton styles={backIconStyles} onClick={_onClickBackToVNetPeeringList} iconProps={backIconProp} />
                </Stack.Item>
                <VNetPeeringsComponent vNetPeerings={props.vNetPeerings} clusterName={props.clusterName} vNetPeeringName={currentVNetPeering}/>
              </Stack>
              :
              <div>
              <ShimmeredDetailsList
                setKey="none"
                items={vNetPeeringsList}
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