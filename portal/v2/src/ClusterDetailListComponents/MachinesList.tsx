import * as React from 'react';
import { useState, useEffect } from "react"
import { Stack, StackItem, IconButton, IIconStyles, SelectionMode } from '@fluentui/react';
import { Link } from '@fluentui/react/lib/Link';
import { IColumn } from '@fluentui/react/lib/DetailsList';
import { ShimmeredDetailsList } from '@fluentui/react/lib/ShimmeredDetailsList';
import { IMachine } from "./MachinesWrapper";
import { MachinesComponent } from "./Machines"



export declare interface IMachinesList {
  name?: string;
  status: string;
  createdTime: string;
}

interface MachinesListComponentProps {
  machines: any
  clusterName: string
}

export interface IMachinesListState {
  machines: IMachine[]
  clusterName: string
}

export class MachinesListComponent extends React.Component<MachinesListComponentProps, IMachinesListState> {
  
  constructor(props: MachinesListComponentProps) {
      super(props)

      this.state = {
          machines: this.props.machines,
          clusterName: this.props.clusterName,
      }
  }
  
  
  public render() {
    return (
        <MachinesListHelperComponent machines={this.state.machines} clusterName={this.state.clusterName}/>
      )
  }
}

export function MachinesListHelperComponent(props: {
     machines: any,
     clusterName: string
}) {
    const [columns, setColumns] = useState<IColumn[]>([
    {
      key: "machineName",
      name: "Name",
      fieldName: "name",
      minWidth: 150,
      maxWidth: 350,
      isResizable: true,
      isSorted: true,
      isSortedDescending: false,
      showSortIconWhenUnsorted: true,
      onRender: (item: IMachinesList) => (
        <Link onClick={() => _onMachineInfoLinkClick(item.name!)}>{item.name}</Link>
      ),
    },
    {
      key: "machineStatus",
      name: "Status",
      fieldName: "status",
      minWidth: 60,
      maxWidth: 60,
      isResizable: true,
      isSorted: true,
      isSortedDescending: false,
      showSortIconWhenUnsorted: true,
    },
    {
      key: "createdTime",
      name: "Created Time",
      fieldName: "createdTime",
      minWidth: 120,
      maxWidth: 150,
      isResizable: true,
      isSorted: true,
      isSortedDescending: false,
      showSortIconWhenUnsorted: true,
    }
  ])

  const [machinesList, setMachinesList] = useState<IMachinesList[]>([])
  const [machinesDetailsVisible, setMachinesDetailsVisible] = useState<boolean>(false)
  const [currentMachine, setCurrentMachine] = useState<string>("")
  const [shimmerVisibility, SetShimmerVisibility] = useState<boolean>(true)


  useEffect(() => {
    setMachinesList(createMachinesList(props.machines))
  }, [props.machines] );


  useEffect(() => {
    const newColumns: IColumn[] = columns.slice();
    newColumns.forEach(col => {
      col.onColumnClick = _onColumnClick
    })
    setColumns(newColumns)

    if (machinesList.length > 0) {
      SetShimmerVisibility(false)
    }
    
  }, [machinesList])

  function _onMachineInfoLinkClick(machine: string) {
    setMachinesDetailsVisible(!machinesDetailsVisible)
    setCurrentMachine(machine)
  }

  function _copyAndSort<T>(items: T[], columnKey: string, isSortedDescending?: boolean): T[] {
    const key = columnKey as keyof T;
    return items.slice(0).sort((a: T, b: T) => ((isSortedDescending ? a[key] < b[key] : a[key] > b[key]) ? 1 : -1));
  }

  function _onColumnClick(event: React.MouseEvent<HTMLElement>, column: IColumn): void {
    let machineLocal: IMachinesList[] = machinesList;
    
    let isSortedDescending = column.isSortedDescending;
    if (column.isSorted) {
      isSortedDescending = !isSortedDescending;
    }

    // Sort the items.
    machineLocal = _copyAndSort(machineLocal, column.fieldName!, isSortedDescending);
    setMachinesList(machineLocal)

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
    //setMachinesList(machineLocal)
    }

    function createMachinesList(machines: IMachine[]): IMachinesList[] {
        return machines.map(machine => {
            return {name: machine.name, status: machine.status, createdTime: machine.createdTime}
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
    setMachinesDetailsVisible(false)
  }

  return (
    <Stack>
      <StackItem>
        {
          machinesDetailsVisible
          ?
          <Stack>
            <Stack.Item>
              <IconButton styles={backIconStyles} onClick={_onClickBackToMachineList} iconProps={backIconProp} />
            </Stack.Item>
            <MachinesComponent machines={props.machines} clusterName={props.clusterName} machineName={currentMachine}/>
          </Stack>
          :
          <div>
          <ShimmeredDetailsList
            setKey="none"
            items={machinesList}
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