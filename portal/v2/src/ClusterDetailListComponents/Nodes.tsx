import { Component } from "react"
import { Stack, Text, StackItem, PivotItem, IStackItemStyles } from "@fluentui/react"
import {
  ICondition,
  INode,
  INodeOverviewDetails,
  IResourceUsage,
  ITaint,
  IVolume,
} from "./NodesWrapper"
import { contentStackStylesNormal } from "../App"
import { InfoList, MultiInfoList } from "./InfoList"

export interface NodesComponentProps {
  nodes: any
  clusterName: string
  nodeName: string
}

const stackItemStyles: IStackItemStyles = {
  root: {
    width: "45%",
  },
}

const NodeOverviewDetails: INodeOverviewDetails = {
  createdTime: "Created Time",
}

const ResourceDetails: IResourceUsage = {
  CPU: "CPU",
  Memory: "Memory",
  StorageVolume: "Storage Volume",
  Pods: "Pods",
}

const ConditionDetails: ICondition = {
  status: "Status",
  lastHeartbeatTime: "Last Heartbeat Time",
  lastTransitionTime: "Last Transition Time",
  message: "Message",
}

const TaintDetails: ITaint = {
  key: "Key",
}

const VolumeDetails: IVolume = {
  Path: "Device Path",
}

export interface INodesState {
  nodes: INode[]
  clusterName: string
  nodeName: string
}

const HeadersFromStringMap = (items: Map<string, string>) => {
  const newItems: any = {}
  items.forEach((value: string, key: string) => {
    newItems[key] = key
  })

  return newItems
}

const ObjectFromStringMap = (items: Map<string, string>) => {
  const newItems: any = {}
  items.forEach((value: string, key: string) => {
    newItems[key] = value
  })

  return newItems
}

const renderNodes = (node: INode) => {
  let length = node.capacity.Memory.length
  node.capacity.Memory =
    Number(Number(node.capacity.Memory.slice(0, length - 2)) / 1048576).toFixed(2) + "Gi"
  length = node.capacity.StorageVolume.length
  node.capacity.StorageVolume =
    Number(Number(node.capacity.StorageVolume.slice(0, length - 2)) / 1048576).toFixed(2) + "Gi"
  return (
    <PivotItem key={node.name} headerText={node.name}>
      <Text variant="xLarge">{node.name}</Text>
      <Stack horizontal grow>
        <Stack styles={stackItemStyles}>
          <StackItem>
            <InfoList
              headers={NodeOverviewDetails}
              object={node}
              title="Overview"
              titleSize="large"
            />
          </StackItem>
          <StackItem>
            <InfoList
              headers={ResourceDetails}
              object={node.capacity}
              title="Capacity"
              titleSize="large"
            />
          </StackItem>
          <StackItem>
            <InfoList
              headers={ResourceDetails}
              object={node.allocatable}
              title="Allocatable"
              titleSize="large"
            />
          </StackItem>
          <StackItem>
            <InfoList
              headers={HeadersFromStringMap(node.labels!)}
              object={ObjectFromStringMap(node.labels!)}
              title="Labels"
              titleSize="large"
            />
          </StackItem>
        </Stack>
        <Stack styles={stackItemStyles}>
          <StackItem>
            <Text variant="large" styles={contentStackStylesNormal}>
              Conditions
            </Text>
            <MultiInfoList
              headers={ConditionDetails}
              items={node.conditions}
              title="Conditions"
              subProp="type"
              titleSize="medium"
            />
          </StackItem>
          <StackItem>
            <Text variant="large" styles={contentStackStylesNormal}>
              Taints
            </Text>
            <MultiInfoList
              headers={TaintDetails}
              items={node.taints}
              title="Taints"
              subProp="effect"
              titleSize="medium"
            />
          </StackItem>
          <StackItem>
            <InfoList
              headers={HeadersFromStringMap(node.annotations!)}
              object={ObjectFromStringMap(node.annotations!)}
              title="Annotations"
              titleSize="large"
            />
          </StackItem>
        </Stack>
        {node.volumes!.length > 0 && (
          <StackItem>
            <Text variant="large">Volumes</Text>
            <MultiInfoList
              headers={VolumeDetails}
              items={node.volumes}
              title="Volumes"
              subProp="Name"
              titleSize="medium"
            />
          </StackItem>
        )}
      </Stack>
    </PivotItem>
  )
}

function Nodes(props: { nodes: any; nodeName: string }) {
  let currentNode: INode

  props.nodes.forEach((node: INode) => {
    if (node.name === props.nodeName) {
      currentNode = node
      return
    }
  })

  return <>{renderNodes(currentNode!)}</>
}
export class NodesComponent extends Component<NodesComponentProps, INodesState> {
  constructor(props: NodesComponentProps) {
    super(props)

    this.state = {
      nodes: this.props.nodes,
      clusterName: this.props.clusterName,
      nodeName: this.props.nodeName,
    }
  }

  public render() {
    return (
      <Stack styles={contentStackStylesNormal}>
        <Stack>
          <Nodes nodes={this.state.nodes} nodeName={this.state.nodeName} />
        </Stack>
      </Stack>
    )
  }
}
