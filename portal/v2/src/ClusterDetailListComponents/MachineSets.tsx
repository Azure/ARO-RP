import { Component } from "react"
import { Stack, StackItem, PivotItem, IStackItemStyles } from "@fluentui/react"
import { contentStackStylesNormal } from "../App"
import { InfoList } from "./InfoList"
import { IMachineSet } from "./MachineSetsWrapper"

interface MachineSetsComponentProps {
  machineSets: any
  clusterName: string
  machineSetName: string
}

const stackItemStyles: IStackItemStyles = {
  root: {
    width: "45%",
  },
}

const MachineSetDetails: IMachineSet = {
  name: "Name",
  type: "Type",
  createdAt: "Created Time",
  desiredReplicas: "Desired Replicas Count",
  replicas: "Actual Replicas Count",
  errorReason: "Error Reason",
  errorMessage: "Error Message",
}

interface IMachineSetsState {
  machineSets: IMachineSet[]
}

const renderMachineSets = (machineSet: IMachineSet) => {
  return (
    <PivotItem key={machineSet.name} headerText={machineSet.name}>
      <Stack styles={stackItemStyles}>
        <StackItem>
          <InfoList
            headers={MachineSetDetails}
            object={machineSet}
            title={machineSet.name!}
            titleSize="large"
          />
        </StackItem>
      </Stack>
    </PivotItem>
  )
}

export class MachineSetsComponent extends Component<MachineSetsComponentProps, IMachineSetsState> {
  constructor(props: MachineSetsComponentProps) {
    super(props)

    this.state = {
      machineSets: this.props.machineSets,
    }
  }

  private extractCurrentMachineSet = (machineSetName: string): IMachineSet => {
    let machineSetTemp: IMachineSet
    this.state.machineSets.forEach((machineSet: IMachineSet) => {
      if (machineSet.name === machineSetName) {
        machineSetTemp = machineSet
        return
      }
    })
    return machineSetTemp!
  }

  public render() {
    return (
      <Stack styles={contentStackStylesNormal}>
        <Stack>{renderMachineSets(this.extractCurrentMachineSet(this.props.machineSetName))}</Stack>
      </Stack>
    )
  }
}
