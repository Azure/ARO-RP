import { useState, useEffect, useRef } from "react"
import { AxiosResponse } from "axios"
import { fetchNodes } from "../Request"
import { ICluster } from "../App"
import { NodesListComponent } from "./NodesList"
import { IMessageBarStyles, MessageBar, MessageBarType, Stack } from "@fluentui/react"
import { nodesKey } from "../ClusterDetail"

export interface ICondition {
  status: string
  lastHeartbeatTime: string
  lastTransitionTime: string
  message: string
}

export interface ITaint {
  key: string
}

export interface IVolume {
  Path: string
}

export interface IResourceUsage {
  CPU: string
  Memory: string
  StorageVolume: string
  Pods: string
}

export interface INode {
  name: string
  createdTime: string
  capacity: IResourceUsage
  allocatable: IResourceUsage
  conditions?: ICondition[]
  taints?: ITaint[]
  labels?: Map<string, string>
  annotations?: Map<string, string>
  volumes?: IVolume[]
}

export interface INodeOverviewDetails {
  createdTime: string
}

export function NodesWrapper(props: {
  currentCluster: ICluster
  detailPanelSelected: string
  loaded: boolean
}) {
  const [data, setData] = useState<any>([])
  const [error, setError] = useState<AxiosResponse | null>(null)
  const state = useRef<NodesListComponent>(null)

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
    const nodeList: INode[] = []
    if (state && state.current) {
      newData.nodes.forEach(
        (element: {
          name: any
          createdTime: any
          capacity: any
          allocatable: any
          taints: ITaint[]
          conditions: ICondition[]
          labels: Record<string, string>
          annotations: Record<string, string>
          volumes: IVolume[]
        }) => {
          const node: INode = {
            name: element.name,
            createdTime: element.createdTime,
            capacity: element.capacity,
            allocatable: element.allocatable,
          }
          node.taints = []
          element.taints.forEach((taint: ITaint) => {
            node.taints!.push(taint)
          })
          node.conditions = []
          element.conditions.forEach((condition: ICondition) => {
            node.conditions!.push(condition)
          })
          node.labels = new Map([])
          Object.entries(element.labels).forEach((label: [string, string]) => {
            node.labels?.set(label[0], label[1])
          })
          node.volumes = []
          element.volumes.forEach((volume: IVolume) => {
            node.volumes!.push(volume)
          })
          node.annotations = new Map([])
          Object.entries(element.annotations).forEach((annotation: [string, string]) => {
            node.annotations?.set(annotation[0], annotation[1])
          })
          nodeList.push(node)
        }
      )
      state.current.setState({ nodes: nodeList })
    }
  }

  useEffect(() => {
    const onData = (result: AxiosResponse | null) => {
      if (result?.status === 200) {
        updateData(result.data)
      } else {
        setError(result)
      }
      setFetching(props.currentCluster.name)
    }

    if (
      props.detailPanelSelected.toLowerCase() == nodesKey &&
      fetching === "" &&
      props.loaded &&
      props.currentCluster.name != ""
    ) {
      setFetching("FETCHING")
      fetchNodes(props.currentCluster).then(onData)
    }
  }, [data, props.loaded, props.detailPanelSelected])

  return (
    <Stack>
      <Stack.Item grow>{error && errorBar()}</Stack.Item>
      <Stack>
        <NodesListComponent
          nodes={data!}
          ref={state}
          clusterName={props.currentCluster != null ? props.currentCluster.name : ""}
        />
      </Stack>
    </Stack>
  )
}
