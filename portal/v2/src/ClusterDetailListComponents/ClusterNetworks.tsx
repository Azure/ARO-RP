import { IStackItemStyles, PivotItem, Stack, StackItem } from "@fluentui/react";
import { Component } from "react";
import { contentStackStylesNormal } from "../App";
import { InfoList } from "./InfoList";
import { IClusterNetwork, IClusterNetworkEntry } from "./NetworkWrapper";


interface ClusterNetworksComponentProps {
    clusterNetworks: any
    clusterName: string
    clusterNetworkName: string
}

const stackItemStyles: IStackItemStyles = {
    root: {
      width: "45%",
    },
  };

export const ClusterNetworkDetails: IClusterNetwork = {
    name: 'Name',
    networkcidr: "Network CIDR",
    servicenetworkcidr: "Service CIDR",
    pluginname: "Plugin Name",
    hostsubnetlength: "Host Subnet Length",
    mtu: "MTU",
    vxlanport: "VXLANPORT"
}

export const ClusterNetworkEntryDetails: IClusterNetworkEntry = {
    cidr: "CIDR",
    hostsubnetlength: "Host Subnet Length"
}

interface IClusterNetworksState {
    clusterNetworks: IClusterNetwork[],
    clusterNetworkName: string,
}

const renderClusterNetworks = (clusterNetwork: IClusterNetwork) => {
    return <PivotItem key={clusterNetwork.name} headerText={clusterNetwork.name}>
                <Stack styles={stackItemStyles} horizontal grow>
                    <StackItem>
                        <InfoList headers={ClusterNetworkDetails} object={clusterNetwork} title={clusterNetwork.name!} titleSize="large"/>
                    </StackItem>
                    <StackItem>
                        <InfoList headers={ClusterNetworkEntryDetails} object={clusterNetwork.clusternerworkentry} title={"Network Entries"} titleSize="large"/>
                    </StackItem>
                </Stack>
        </PivotItem>;
}

function PivotOverflowMenuExample(props: {
    clusterNetworks: any,
    clusterNetworkName: string
}) {   
    let currentClusterNetwork: IClusterNetwork
    
    props.clusterNetworks.forEach((clusterNetwork: IClusterNetwork) => {
        if (clusterNetwork.name === props.clusterNetworkName) {
            currentClusterNetwork = clusterNetwork
            return
        }
    })
    
    return (
            <>
                {renderClusterNetworks(currentClusterNetwork!)}
            </>
    );
}

export class ClusterNetworksComponent extends Component<ClusterNetworksComponentProps, IClusterNetworksState> {

    constructor(props: ClusterNetworksComponentProps) {
        super(props)

        this.state = {
            clusterNetworks: this.props.clusterNetworks,
            clusterNetworkName: this.props.clusterNetworkName,
        }
    }

    // private extractCurrentClusterNetwork = (clusterNetworkName: string): IClusterNetwork => {
    //     this.state.clusterNetworks.forEach((clusterNetwork: IClusterNetwork) => {
    //         if (clusterNetwork.name === clusterNetworkName) {
    //             return clusterNetwork
    //         }
    //     })
    //     return this.state.clusterNetworks[0]
    // }

    public render() {
        return (
        <Stack styles={contentStackStylesNormal}>
            <Stack>
            <PivotOverflowMenuExample clusterNetworks={this.state.clusterNetworks} clusterNetworkName={this.state.clusterNetworkName}/>
            </Stack>
        </Stack>
        )
    }
}