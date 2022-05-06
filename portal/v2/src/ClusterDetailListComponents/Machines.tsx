import { Component } from "react"
import { Stack, Text, StackItem, Pivot, PivotItem, IStackItemStyles, } from '@fluentui/react';
import { contentStackStylesNormal } from "../App";
import { InfoList } from "./InfoList"
import { IMachine } from "./MachinesWrapper";

interface MachinesComponentProps {
    machines: any
    clusterName: string
}

const stackItemStyles: IStackItemStyles = {
    root: {
      width: "45%",
    },
  };

const MachineDetails: IMachine = {
    createdTime: 'Created Time',
    lastUpdated: "Last Updated",
    errorReason: "Error Reason",
    errorMessage: "Error Message",
    lastOperation: "Last Operation",
    lastOperationDate: "Last Operation Date",
    status: "Status"
}

interface IMachinesState {
machines: IMachine[]
}

const renderMachines = (machines: IMachine[]) => {
    return machines.map(machine => {
        return <PivotItem key={machine.name} headerText={machine.name}>
                    <Stack styles={stackItemStyles}>
                        <StackItem>
                            <InfoList headers={MachineDetails} object={machine} title={machine.name!} titleSize="large"/>
                        </StackItem>
                    </Stack>
            </PivotItem>;
    });
};

export class MachinesComponent extends Component<MachinesComponentProps, IMachinesState> {

    constructor(props: MachinesComponentProps) {
        super(props)

        this.state = {
            machines: this.props.machines,
        }
    }

    public render() {
        return (
        <Stack styles={contentStackStylesNormal}>
            <Text variant="xxLarge">{this.props.clusterName}</Text>
            <Stack>
                <Pivot linkFormat={'tabs'} overflowBehavior={'menu'}>
                    {renderMachines(this.state.machines)}
                </Pivot>
            </Stack>
        </Stack>
        )
    }
}
