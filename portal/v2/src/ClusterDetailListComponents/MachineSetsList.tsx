import * as React from 'react';
import { useState, useEffect } from "react"
import { Stack, StackItem, IconButton, IIconStyles } from '@fluentui/react';
import { Link } from '@fluentui/react/lib/Link';
import { DetailsList, IColumn,IDetailsListStyles } from '@fluentui/react/lib/DetailsList';
import { IMachineSet } from "./MachineSetsWrapper";
import { MachineSetsComponent } from "./MachineSets"



export declare interface IMachineSetsList {
  name?: string;
  desiredReplicas: string;
  currentReplicas: string;
  publicLoadBalancer?: string;
  storageType?: string
}

interface MachineSetsListComponentProps {
  machineSets: any
  clusterName: string
}

export interface IMachineSetsListState {
  machineSets: IMachineSet[]
  clusterName: string
}

export class MachineSetsListComponent extends React.Component<MachineSetsListComponentProps, IMachineSetsListState> {
  
  constructor(props: MachineSetsListComponentProps) {
      super(props)

      this.state = {
          machineSets: this.props.machineSets,
          clusterName: this.props.clusterName,
      }      
  }
  
  
  public render() {
    return (
        <MachineSetsListHelperComponent machineSets={this.state.machineSets} clusterName={this.state.clusterName}/>
      )
  }
}

export function MachineSetsListHelperComponent(props: {
     machineSets: any,
     clusterName: string
}) {
    const [columns, setColumns] = useState<IColumn[]>([
    {
      key: "machineName",
      name: "Name",
      fieldName: "name",
      minWidth: 100,
      maxWidth: 200,
      isResizable: true,
      isSorted: true,
      isSortedDescending: false,
      showSortIconWhenUnsorted: true,
      onRender: (item: IMachineSetsList) => (
        <Link onClick={() => _onMachineInfoLinkClick(item.name!)}>{item.name}</Link>
      ),
    },
    {
      key: "desiredReplicas",
      name: "Desired Replicas",
      fieldName: "desiredReplicas",
      minWidth: 120,
      maxWidth: 200,
      isResizable: true,
      isSorted: true,
      isSortedDescending: false,
      showSortIconWhenUnsorted: true,
    },
    {
      key: "currentReplicas",
      name: "Current Replicas",
      fieldName: "currentReplicas",
      minWidth: 120,
      maxWidth: 200,
      isResizable: true,
      isSorted: true,
      isSortedDescending: false,
      showSortIconWhenUnsorted: true,
    },
    {
      key: "publicLoadBalancer",
      name: "Public LoadBalancer",
      fieldName: "publicLoadBalancer",
      minWidth: 150,
      maxWidth: 200,
      isResizable: true,
      isSorted: true,
      isSortedDescending: false,
      showSortIconWhenUnsorted: true,
    },
    {
      key: "storageType",
      name: "Storage Type",
      fieldName: "storageType",
      minWidth: 100,
      maxWidth: 200,
      isResizable: true,
      isSorted: true,
      isSortedDescending: false,
      showSortIconWhenUnsorted: true,
    }
  ])

  const [machineSetsList, setMachineSetsList] = useState<IMachineSetsList[]>([])
  const [machineSetsDetailsVisible, setMachineSetsDetailsVisible] = useState<boolean>(false)
  const [currentMachine, setCurrentMachine] = useState<string>("")

  useEffect(() => {
    setMachineSetsList(createMachineSetsList(props.machineSets))
  }, [props.machineSets] );

  useEffect(() => {
    const newColumns: IColumn[] = columns.slice();
    newColumns.forEach(col => {
      col.onColumnClick = _onColumnClick
    })
    setColumns(newColumns)
  }, [machineSetsList])

  function _onMachineInfoLinkClick(machine: string) {
    setMachineSetsDetailsVisible(!machineSetsDetailsVisible)
    setCurrentMachine(machine)
  }

  function _copyAndSort<T>(items: T[], columnKey: string, isSortedDescending?: boolean): T[] {
    const key = columnKey as keyof T;
    return items.slice(0).sort((a: T, b: T) => ((isSortedDescending ? a[key] < b[key] : a[key] > b[key]) ? 1 : -1));
  }

  function _onColumnClick(event: React.MouseEvent<HTMLElement>, column: IColumn): void {
    let machineLocal: IMachineSetsList[] = machineSetsList;
    
    let isSortedDescending = column.isSortedDescending;
    if (column.isSorted) {
      isSortedDescending = !isSortedDescending;
    }

    // Sort the items.
    machineLocal = _copyAndSort(machineLocal, column.fieldName!, isSortedDescending);
    setMachineSetsList(machineLocal)

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

    function _getKey(item: any): string {
        return item.key
    }

    function createMachineSetsList(MachineSets: IMachineSet[]): IMachineSetsList[] {
        return MachineSets.map(machineSet => {
            return {name: machineSet.name, desiredReplicas: machineSet.desiredReplicas!, currentReplicas: machineSet.replicas!, publicLoadBalancer: machineSet.publicLoadBalancerName, storageType: machineSet.accountStorageType}
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
  function _onClickBackToMachineList() {
    setMachineSetsDetailsVisible(false)
  }

  const gridStyles: Partial<IDetailsListStyles> = {
    root: {
      overflowX: 'scroll',
      selectors: {
        '& [role=grid]': {
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'start',
          height: '60vh',
        },
      },
    },
    headerWrapper: {
      flex: '0 0 auto',
    },
    contentWrapper: {
      flex: '1 1 auto',
      overflowY: 'auto',
      overflowX: 'hidden',
    },
  };

  return (
    <Stack>
      <StackItem>
        {
          machineSetsDetailsVisible
          ?
          <Stack>
            <Stack.Item>
              <IconButton styles={backIconStyles} onClick={_onClickBackToMachineList} iconProps={backIconProp} />
            </Stack.Item>
            <MachineSetsComponent machineSets={props.machineSets} clusterName={props.clusterName} machineSetName={currentMachine}/>
          </Stack>
          :
          <DetailsList
              items={machineSetsList}
              setKey="none"
              columns={columns}
              selectionMode={0}
              getKey={_getKey}
              constrainMode={0}
              styles={gridStyles}
              ariaLabelForSelectionColumn="Toggle selection"
              ariaLabelForSelectAllCheckbox="Toggle selection for all items"
              checkButtonAriaLabel="select row"
            />    
        }
      </StackItem>
    </Stack>
  )
}