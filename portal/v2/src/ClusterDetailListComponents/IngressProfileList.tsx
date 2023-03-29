import { IColumn, IconButton, IIconStyles, Link, SelectionMode, ShimmeredDetailsList, Stack, StackItem } from '@fluentui/react';
import * as React from 'react';
import { Component, useEffect, useState } from 'react';
import { IngressProfilesComponent } from './IngressProfiles';
import { IIngressProfile } from './NetworkWrapper';


export declare interface IIngressProfilesList {
    name?: string;
    ip: string;
    visibility: string;
}
  
interface IngressProfilesListComponentProps {
    ingressProfiles: any
    clusterName: string
}
  
export interface IIngressProfilesListState {
    ingressProfiles: IIngressProfile[]
    clusterName: string
}

export class IngressProfilesListComponent extends Component<IngressProfilesListComponentProps, IIngressProfilesListState> {

    constructor(props: IngressProfilesListComponentProps) {
        super(props)

        this.state = {
            ingressProfiles: this.props.ingressProfiles,
            clusterName: this.props.clusterName,
        }
    }

    public render() {
        return (
            <IngressProfilesListHelperComponent ingressProfiles={this.state.ingressProfiles} clusterName={this.state.clusterName}/>
          )
    }
}

export function IngressProfilesListHelperComponent(props: {
    ingressProfiles: any,
    clusterName: string
}) {
    const [columns, setColumns] = useState<IColumn[]>([
        {
            key: "ingressProfileName",
            name: "Name",
            fieldName: "name",
            minWidth: 150,
            maxWidth: 350,
            isResizable: true,
            isSorted: true,
            isSortedDescending: false,
            showSortIconWhenUnsorted: true,
            onRender: (item: IIngressProfilesList) => (
            <Link onClick={() => _onIngressProfileInfoLinkClick(item.name!)}>{item.name}</Link>
            ),
        },
        {
            key: "ingressProfileIp",
            name: "IP Address",
            fieldName: "ip",
            minWidth: 60,
            maxWidth: 120,
            isResizable: true,
            isSorted: true,
            isSortedDescending: false,
            showSortIconWhenUnsorted: true,
        },
        {
            key: "ingressProfileVisibility",
            name: "Visibility",
            fieldName: "visibility",
            minWidth: 60,
            maxWidth: 80,
            isResizable: true,
            isSorted: true,
            isSortedDescending: false,
            showSortIconWhenUnsorted: true,
        }
    ])

    const [ingressProfilesList, setIngressProfilesList] = useState<IIngressProfilesList[]>([])
    const [ingressProfilesDetailsVisible, setIngressProfilesDetailsVisible] = useState<boolean>(false)
    const [currentIngressProfile, setCurrentIngressProfile] = useState<string>("")
    const [shimmerVisibility, SetShimmerVisibility] = useState<boolean>(true)

    useEffect(() => {
        setIngressProfilesList(createIngressProfilesList(props.ingressProfiles))
    }, [props.ingressProfiles] )

    useEffect(() => {
        const newColumns: IColumn[] = columns.slice();
        newColumns.forEach(col => {
        col.onColumnClick = _onColumnClick
        })
        setColumns(newColumns)

        if (ingressProfilesList.length > 0) {
        SetShimmerVisibility(false)
        }
        
    }, [ingressProfilesList])

    function _onIngressProfileInfoLinkClick(ingressProfile: string) {
        setIngressProfilesDetailsVisible(!ingressProfilesDetailsVisible)
        setCurrentIngressProfile(ingressProfile)
    }

    function _copyAndSort<T>(items: T[], columnKey: string, isSortedDescending?: boolean): T[] {
        const key = columnKey as keyof T;
        return items.slice(0).sort((a: T, b: T) => ((isSortedDescending ? a[key] < b[key] : a[key] > b[key]) ? 1 : -1));
    }

    function _onColumnClick(event: React.MouseEvent<HTMLElement>, column: IColumn): void {
        let ingressProfileLocal: IIngressProfilesList[] = ingressProfilesList;
        
        let isSortedDescending = column.isSortedDescending;
        if (column.isSorted) {
            isSortedDescending = !isSortedDescending;
        }

        // Sort the items.
        ingressProfileLocal = _copyAndSort(ingressProfileLocal, column.fieldName!, isSortedDescending);
        setIngressProfilesList(ingressProfileLocal)

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

    function createIngressProfilesList(ingressProfiles: IIngressProfile[]): IIngressProfilesList[] {
        return ingressProfiles.map(ingressProfile => {
            return {name: ingressProfile.name, ip: ingressProfile.ip, visibility: ingressProfile.visibility}
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
      function _onClickBackToIngressProfileList() {
        setIngressProfilesDetailsVisible(false)
    }

    return (
        <Stack>
          <StackItem>
            {
              ingressProfilesDetailsVisible
              ?
              <Stack>
                <Stack.Item>
                  <IconButton styles={backIconStyles} onClick={_onClickBackToIngressProfileList} iconProps={backIconProp} />
                </Stack.Item>
                <IngressProfilesComponent ingressProfiles={props.ingressProfiles} clusterName={props.clusterName} ingressProfileName={currentIngressProfile}/>
              </Stack>
              :
              <div>
              <ShimmeredDetailsList
                setKey="none"
                items={ingressProfilesList}
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