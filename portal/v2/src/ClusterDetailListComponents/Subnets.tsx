import { IStackItemStyles, PivotItem, Stack, StackItem } from "@fluentui/react";
import { Component } from "react";
import { contentStackStylesNormal } from "../App";
import { InfoList } from "./InfoList";
import { ISubnet } from "./NetworkWrapper";


interface SubnetsComponentProps {
    subnets: any
    clusterName: string
    subnetName: string
}

const stackItemStyles: IStackItemStyles = {
    root: {
      width: "45%",
    },
  };

export const SubnetDetails: ISubnet = {
    name: 'Name',
    addressprefix: "Address Prefix",
    provisioning: "Provisioning",
    routetable: "Route Table",
    id: "ID"
}

interface ISubnetsState {
    subnets: ISubnet[],
    subnetName: string,
}

const renderSubnets = (subnet: ISubnet) => {
    return <PivotItem key={subnet.name} headerText={subnet.name}>
                <Stack styles={stackItemStyles}>
                    <StackItem>
                        <InfoList headers={SubnetDetails} object={subnet} title={subnet.name!} titleSize="large"/>
                    </StackItem>
                </Stack>
        </PivotItem>;
}

export class SubnetsComponent extends Component<SubnetsComponentProps, ISubnetsState> {

    constructor(props: SubnetsComponentProps) {
        super(props)

        this.state = {
            subnets: this.props.subnets,
            subnetName: this.props.subnetName,
        }
    }

    private extractCurrentSubnet = (subnetName: string): ISubnet => {
        this.state.subnets.forEach((subnet: ISubnet) => {
            if (subnet.name === subnetName) {
                return subnet
            }
        })
        return this.state.subnets[0]
    }

    public render() {
        return (
        <Stack styles={contentStackStylesNormal}>
            <Stack>
                {renderSubnets(this.extractCurrentSubnet(this.state.subnetName))}
            </Stack>
        </Stack>
        )
    }
}