import { IStackItemStyles, PivotItem, Stack, StackItem } from "@fluentui/react";
import { Component } from "react";
import { contentStackStylesNormal } from "../App";
import { InfoList } from "./InfoList";
import { IVNetPeering } from "./NetworkWrapper";


interface VNetPeeringsComponentProps {
    vNetPeerings: any
    clusterName: string
    vNetPeeringName: string
}

const stackItemStyles: IStackItemStyles = {
    root: {
      width: "45%",
    },
  };

export const VNetPeeringDetails: IVNetPeering = {
    name: 'Name',
    remotevnet: "Remote VNet",
    state: "State",
    provisioning: "Provisioning",
}

interface IVNetPeeringsState {
    vNetPeerings: IVNetPeering[],
    vNetPeeringName: string,
}

const renderVNetPeerings = (vNetPeering: IVNetPeering) => {
    return <PivotItem key={vNetPeering.name} headerText={vNetPeering.name}>
                <Stack styles={stackItemStyles}>
                    <StackItem>
                        <InfoList headers={VNetPeeringDetails} object={vNetPeering} title={vNetPeering.name!} titleSize="large"/>
                    </StackItem>
                </Stack>
        </PivotItem>;
}

export class VNetPeeringsComponent extends Component<VNetPeeringsComponentProps, IVNetPeeringsState> {

    constructor(props: VNetPeeringsComponentProps) {
        super(props)

        this.state = {
            vNetPeerings: this.props.vNetPeerings,
            vNetPeeringName: this.props.vNetPeeringName,
        }
    }

    private extractCurrentVNetPeering = (vNetPeeringName: string): IVNetPeering => {
        this.state.vNetPeerings.forEach((vNetPeering: IVNetPeering) => {
            if (vNetPeering.name === vNetPeeringName) {
                return vNetPeering
            }
        })
        return this.state.vNetPeerings[0]
    }

    public render() {
        return (
        <Stack styles={contentStackStylesNormal}>
            <Stack>
                {renderVNetPeerings(this.extractCurrentVNetPeering(this.state.vNetPeeringName))}
            </Stack>
        </Stack>
        )
    }
}