import { Component } from "react"
import { Stack, Text, PivotItem, IStackItemStyles } from "@fluentui/react"
import { ICondition, IClusterOperator } from "./ClusterOperatorsWrapper"
import { contentStackStylesNormal } from "../App"
import { MultiInfoList } from "./InfoList"

export interface ClusterOperatorComponentProps {
  clusterOperators: IClusterOperator[]
  clusterName: string
  clusterOperatorName: string
}

const stackItemStyles: IStackItemStyles = {
  root: {
    width: "100%",
  },
}

const ConditionDetails: ICondition = {
  status: "Status",
  reason: "Reason",
  lastUpdated: "LastUpdated",
  message: "Message",
}

export interface IClusterOperatorsState {
  clusterOperators: IClusterOperator[]
  clusterName: string
  clusterOperatorName: string
}

const renderClusterOperators = (operator: IClusterOperator) => {
  return (
    <PivotItem key={operator.name} headerText={operator.name}>
      <Stack>
        <Text variant="xLarge">{operator.name}</Text>
        <Text variant="large" styles={contentStackStylesNormal}>
          Conditions
        </Text>
      </Stack>
      <Stack wrap styles={stackItemStyles} horizontal grow>
        <MultiInfoList
          headers={ConditionDetails}
          items={operator.conditions}
          title="Conditions"
          subProp="type"
          titleSize="medium"
        />
      </Stack>
    </PivotItem>
  )
}

function Operators(props: { clusterOperators: IClusterOperator[]; clusterOperatorName: string }) {
  for (const operator of props.clusterOperators) {
    if (operator.name === props.clusterOperatorName) {
      return <>{renderClusterOperators(operator)}</>
    }
  }
  return <></>
}
export class ClusterOperatorsComponent extends Component<
  ClusterOperatorComponentProps,
  IClusterOperatorsState
> {
  constructor(props: ClusterOperatorComponentProps) {
    super(props)

    this.state = {
      clusterOperators: this.props.clusterOperators,
      clusterName: this.props.clusterName,
      clusterOperatorName: this.props.clusterOperatorName,
    }
  }

  public render() {
    return (
      <Stack styles={contentStackStylesNormal}>
        <Stack>
          <Operators
            clusterOperators={this.state.clusterOperators}
            clusterOperatorName={this.state.clusterOperatorName}
          />
        </Stack>
      </Stack>
    )
  }
}
