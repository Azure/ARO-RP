import { Component } from "react"
import { Stack, Text, StackItem, Pivot, PivotItem, IStackItemStyles, } from '@fluentui/react';
import { contentStackStylesNormal } from "../App";
import { InfoList } from "./InfoList"
import { IMachineSet } from "./MachineSetsWrapper";

interface MachineSetsComponentProps {
    machineSets: any
    clusterName: string
}

const stackItemStyles: IStackItemStyles = {
    root: {
      width: "45%",
    },
  };

const MachineSetDetails: IMachineSet = {
    name: 'Name',
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

// TODO: Get Styling to look pretty
const renderMachineSets = (machineSets: IMachineSet[]) => {
    return machineSets.map(machineSet => {
        return <PivotItem key={machineSet.name} headerText={machineSet.name}>
                    <Stack styles={stackItemStyles}>
                        <StackItem>
                            <InfoList headers={MachineSetDetails} object={machineSet} title={machineSet.name!} titleSize="large"/>
                        </StackItem>
                    </Stack>
            </PivotItem>;
    });
};

export class MachineSetsComponent extends Component<MachineSetsComponentProps, IMachineSetsState> {

    constructor(props: MachineSetsComponentProps) {
        super(props)

        this.state = {
            machineSets: this.props.machineSets,
        }
    }

    public render() {
        return (
        <Stack styles={contentStackStylesNormal}>
            <Text variant="xxLarge">{this.props.clusterName}</Text>
            <Stack>
                <Pivot linkFormat={'tabs'} overflowBehavior={'menu'}>
                    {renderMachineSets(this.state.machineSets)}
                </Pivot>
            </Stack>
        </Stack>
        )
    }
}
