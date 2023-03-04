import { IStackItemStyles, PivotItem, Stack, StackItem } from "@fluentui/react";
import { Component } from "react";
import { contentStackStylesNormal } from "../App";
import { InfoList } from "./InfoList";
import { IIngressProfile } from "./NetworkWrapper";


interface IngressProfilesComponentProps {
    ingressProfiles: any
    clusterName: string
    ingressProfileName: string
}

const stackItemStyles: IStackItemStyles = {
    root: {
      width: "45%",
    },
  };

export const IngressProfileDetails: IIngressProfile = {
    name: 'Name',
    ip: "IP Address",
    visibility: "Visibility"
}

interface IIngressProfilesState {
    ingressProfiles: IIngressProfile[],
    ingressProfileName: string,
}

const renderIngressProfiles = (ingressProfile: IIngressProfile) => {
    return <PivotItem key={ingressProfile.name} headerText={ingressProfile.name}>
                <Stack styles={stackItemStyles}>
                    <StackItem>
                        <InfoList headers={IngressProfileDetails} object={ingressProfile} title={ingressProfile.name!} titleSize="large"/>
                    </StackItem>
                </Stack>
        </PivotItem>;
}

export class IngressProfilesComponent extends Component<IngressProfilesComponentProps, IIngressProfilesState> {

    constructor(props: IngressProfilesComponentProps) {
        super(props)

        this.state = {
            ingressProfiles: this.props.ingressProfiles,
            ingressProfileName: this.props.ingressProfileName,
        }
    }

    private extractCurrentIngressProfile = (ingressProfileName: string): IIngressProfile => {
        this.state.ingressProfiles.forEach((ingressProfile: IIngressProfile) => {
            if (ingressProfile.name === ingressProfileName) {
                return ingressProfile
            }
        })
        return this.state.ingressProfiles[0]
    }

    public render() {
        return (
        <Stack styles={contentStackStylesNormal}>
            <Stack>
                {renderIngressProfiles(this.extractCurrentIngressProfile(this.state.ingressProfileName))}
            </Stack>
        </Stack>
        )
    }
}