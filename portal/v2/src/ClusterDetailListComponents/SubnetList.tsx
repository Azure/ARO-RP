import { IColumn, IconButton, IIconStyles, Link, SelectionMode, ShimmeredDetailsList, Stack, StackItem } from '@fluentui/react';
import * as React from 'react';
import { useEffect, useState } from 'react';
import { ISubnet } from "./NetworkWrapper";
import { SubnetsComponent } from './Subnets';


export declare interface ISubnetsList {
    name?: string;
    addressprefix: string;
    provisioning: string;
}
  
interface SubnetsListComponentProps {
    subnets: any
    clusterName: string
}
  
export interface ISubnetsListState {
    subnets: ISubnet[]
    clusterName: string
}

export class SubnetsListComponent extends React.Component<SubnetsListComponentProps, ISubnetsListState> {

    constructor(props: SubnetsListComponentProps) {
        super(props)

        this.state = {
            subnets: this.props.subnets,
            clusterName: this.props.clusterName,
        }
    }

    public render() {
        return (
            <SubnetsListHelperComponent subnets={this.state.subnets} clusterName={this.state.clusterName}/>
          )
    }
}

export function SubnetsListHelperComponent(props: {
    subnets: any,
    clusterName: string
}) {
    const [columns, setColumns] = useState<IColumn[]>([
        {
            key: "subnetName",
            name: "Name",
            fieldName: "name",
            minWidth: 150,
            maxWidth: 350,
            isResizable: true,
            isSorted: true,
            isSortedDescending: false,
            showSortIconWhenUnsorted: true,
            onRender: (item: ISubnetsList) => (
            <Link onClick={() => _onSubnetInfoLinkClick(item.name!)}>{item.name}</Link>
            ),
        },
        {
            key: "subnetAddressPrefix",
            name: "Address Prefix",
            fieldName: "addressprefix",
            minWidth: 60,
            maxWidth: 120,
            isResizable: true,
            isSorted: true,
            isSortedDescending: false,
            showSortIconWhenUnsorted: true,
        },
        {
            key: "subnetProvisioning",
            name: "Provisioning",
            fieldName: "provisioning",
            minWidth: 60,
            maxWidth: 80,
            isResizable: true,
            isSorted: true,
            isSortedDescending: false,
            showSortIconWhenUnsorted: true,
        }
    ])

    const [subnetsList, setSubnetsList] = useState<ISubnetsList[]>([])
    const [subnetsDetailsVisible, setSubnetsDetailsVisible] = useState<boolean>(false)
    const [currentSubnet, setCurrentSubnet] = useState<string>("")
    const [shimmerVisibility, SetShimmerVisibility] = useState<boolean>(true)

    useEffect(() => {
        setSubnetsList(createSubnetsList(props.subnets))
    }, [props.subnets] )

    useEffect(() => {
        const newColumns: IColumn[] = columns.slice();
        newColumns.forEach(col => {
        col.onColumnClick = _onColumnClick
        })
        setColumns(newColumns)

        if (subnetsList.length > 0) {
        SetShimmerVisibility(false)
        }
        
    }, [subnetsList])

    function _onSubnetInfoLinkClick(subnet: string) {
        setSubnetsDetailsVisible(!subnetsDetailsVisible)
        setCurrentSubnet(subnet)
    }

    function _copyAndSort<T>(items: T[], columnKey: string, isSortedDescending?: boolean): T[] {
        const key = columnKey as keyof T;
        return items.slice(0).sort((a: T, b: T) => ((isSortedDescending ? a[key] < b[key] : a[key] > b[key]) ? 1 : -1));
    }

    function _onColumnClick(event: React.MouseEvent<HTMLElement>, column: IColumn): void {
        let subnetLocal: ISubnetsList[] = subnetsList;
        
        let isSortedDescending = column.isSortedDescending;
        if (column.isSorted) {
            isSortedDescending = !isSortedDescending;
        }

        // Sort the items.
        subnetLocal = _copyAndSort(subnetLocal, column.fieldName!, isSortedDescending);
        setSubnetsList(subnetLocal)

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

    function createSubnetsList(subnets: ISubnet[]): ISubnetsList[] {
        return subnets.map(subnet => {
            return {name: subnet.name, addressprefix: subnet.addressprefix, provisioning: subnet.provisioning}
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
      function _onClickBackToSubnetList() {
        setSubnetsDetailsVisible(false)
    }

    return (
        <Stack>
          <StackItem>
            {
              subnetsDetailsVisible
              ?
              <Stack>
                <Stack.Item>
                  <IconButton styles={backIconStyles} onClick={_onClickBackToSubnetList} iconProps={backIconProp} />
                </Stack.Item>
                <SubnetsComponent subnets={props.subnets} clusterName={props.clusterName} subnetName={currentSubnet}/>
              </Stack>
              :
              <div>
              <ShimmeredDetailsList
                setKey="none"
                items={subnetsList}
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
