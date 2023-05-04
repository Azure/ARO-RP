import { Component } from "react"
import { Stack, StackItem, PivotItem, IStackItemStyles, } from '@fluentui/react';
import { contentStackStylesNormal } from "../App";
import { InfoList } from "./InfoList"
import { IMachine } from "./MachinesWrapper";

interface MachinesComponentProps {
    machines: any
    clusterName: string
    machineName: string
}

const stackItemStyles: IStackItemStyles = {
    root: {
      width: "45%",
    },
  };

export const MachineDetails: IMachine = {
    createdTime: 'Created Time',
    lastUpdated: "Last Updated",
    errorReason: "Error Reason",
    errorMessage: "Error Message",
    lastOperation: "Last Operation",
    lastOperationDate: "Last Operation Date",
    status: "Status"
}

interface IMachinesState {
    machines: IMachine[],
    machineName: string,
}

const renderMachines = (machine: IMachine) => {
    return <PivotItem key={machine.name} headerText={machine.name}>
                <Stack styles={stackItemStyles}>
                    <StackItem>
                        <InfoList headers={MachineDetails} object={machine} title={machine.name!} titleSize="large"/>
                    </StackItem>
                </Stack>
        </PivotItem>;
};

export class MachinesComponent extends Component<MachinesComponentProps, IMachinesState> {

    constructor(props: MachinesComponentProps) {
        super(props)

        this.state = {
            machines: this.props.machines,
            machineName: this.props.machineName,
        }
    }

    private extractCurrentMachine = (machineName: string): IMachine => {
        this.state.machines.forEach((machine: IMachine) => {
            if (machine.name === machineName) {
                return machine
            }
        })
        return this.state.machines[0]
    }

    public render() {
        return (
        <Stack styles={contentStackStylesNormal}>
            <Stack>
                {renderMachines(this.extractCurrentMachine(this.state.machineName))}
            </Stack>
        </Stack>
        )
    }
}
