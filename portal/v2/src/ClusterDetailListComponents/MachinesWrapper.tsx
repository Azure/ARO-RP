import { useState, useEffect, useRef } from "react"
import { fetchMachines } from "../Request"
import { MachinesListComponent } from "./MachinesList"
import {
  IMessageBarStyles,
  MessageBar,
  MessageBarType,
  Stack,
  CommandBar,
  ICommandBarItemProps,
} from "@fluentui/react"
import { machinesKey } from "../ClusterDetail"
import { WrapperProps } from "../ClusterDetailList"

export interface IMachine {
  name?: string
  createdTime: string
  lastUpdated: string
  errorReason: string
  errorMessage: string
  lastOperation: string
  lastOperationDate: string
  status: string
}

export function MachinesWrapper(props: WrapperProps) {
  const [data, setData] = useState<any>([])
  const [error, setError] = useState<Response | null>(null)
  const state = useRef<MachinesListComponent>(null)
  const [fetching, setFetching] = useState("")

  const errorBarStyles: Partial<IMessageBarStyles> = { root: { marginBottom: 15 } }

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

  // updateData - updates the state of the component
  // can be used if we want a refresh button.
  // api/clusterdetail returns a single item.
  const updateData = (newData: any) => {
    setData(newData)
    const machineList: IMachine[] = []
    if (state && state.current) {
      newData.machines?.forEach(
        (element: {
          name: string
          createdTime: string
          lastUpdated: string
          errorReason: string
          errorMessage: string
          lastOperation: string
          lastOperationDate: string
          status: string
        }) => {
          const machine: IMachine = {
            name: element.name,
            createdTime: element.createdTime,
            lastUpdated: element.lastUpdated,
            errorReason: element.errorReason,
            errorMessage: element.errorMessage,
            lastOperation: element.lastOperation,
            lastOperationDate: element.lastOperationDate,
            status: element.status,
          }
          machineList.push(machine)
        }
      )
      state.current.setState({ machines: machineList })
    }
  }

  const controlStyles = {
    root: {
      paddingLeft: 0,
      float: "right",
    },
  }

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

  useEffect(() => {
    const onData = async (result: Response) => {
      if (result.status === 200) {
        updateData(await result.json())
      } else {
        setError(result)
      }
      if (props.currentCluster) {
        setFetching(props.currentCluster.name)
      }
    }

    if (
      props.detailPanelSelected.toLowerCase() == machinesKey &&
      fetching === "" &&
      props.loaded &&
      props.currentCluster
    ) {
      setFetching("FETCHING")
      fetchMachines(props.currentCluster).then(onData)
    }
  }, [data, props.loaded, props.detailPanelSelected])

  return (
    <Stack>
      <Stack.Item grow>{error && errorBar()}</Stack.Item>
      <Stack>
        <CommandBar items={_items} ariaLabel="Refresh" styles={controlStyles} />
        <MachinesListComponent
          machines={data!}
          ref={state}
          clusterName={props.currentCluster != null ? props.currentCluster.name : ""}
        />
      </Stack>
    </Stack>
  )
}
