import { IShimmerStyles, Shimmer, ShimmerElementType } from '@fluentui/react/lib/Shimmer';
import { Component } from "react"
import { Stack, Text, IStackStyles, IStackItemStyles } from '@fluentui/react';
import { contentStackStylesNormal } from "../App"
import { IClusterDetails } from "../ClusterDetailList"

interface OverviewComponentProps {
    item: any
    clusterName: string
}

interface IOverviewComponentState {
    item: IClusterDetails
}

export const ShimmerStyle: Partial<IShimmerStyles> = {
    root: {
        margin: "11px 0"
    }
}

export const headShimmerStyle: Partial<IShimmerStyles> = {
    root: {
        margin: "15px 0"
    }
}

export const headerShimmer = [
    { type: ShimmerElementType.line, height: 32, width: '25%' },
]

export const rowShimmer = [
    { type: ShimmerElementType.line, height: 18, width: '75%' },
]

export const KeyColumnStyle: Partial<IStackStyles> = {
    root: {
        paddingTop: 10,
        paddingRight: 15,
    }
}

export const ValueColumnStyle: Partial<IStackStyles> = {
    root: {
        paddingTop: 10,
    }
}

export const KeyStyle: IStackItemStyles = {
    root: {
        fontStyle: "bold",
        alignSelf: "flex-start",
        fontVariantAlternates: "bold",
        color: "grey",
        paddingBottom: 10
    }
}

export const ValueStyle: IStackItemStyles = {
    root: {
        paddingBottom: 10
    }
}

const clusterDetailHeadings : IClusterDetails = {
    apiServerVisibility: 'ApiServer Visibility',
    apiServerURL: 'ApiServer URL',
    architectureVersion: 'Architecture Version',
    consoleLink: 'Console Link',
    createdAt: 'Created At',
    createdBy: 'Created By',
    failedProvisioningState: 'Failed Provisioning State',
    infraId: 'Infra Id',
    lastAdminUpdateError: 'Last Admin Update Error',
    lastModifiedAt: 'Last Modified At',
    lastModifiedBy: 'Last Modified By',
    lastProvisioningState: 'Last Provisioning State',
    location: 'Location',
    name: 'Name',
    provisioningState: 'Provisioning State',
    resourceId: 'Resource Id',
    version: 'Version',
    installStatus: 'Installation Status'
}

function ClusterDetailCell(
    value: any,
    ): any {
        if (typeof (value.value) == typeof (" ")) {
            return <Stack.Item id="ClusterDetailCell" styles={value.style}>
            <Text styles={value.style} variant={'medium'}>{value.value}</Text>
            </Stack.Item>
        }
    }
    
export class OverviewComponent extends Component<OverviewComponentProps, IOverviewComponentState> {
    
    constructor(props: OverviewComponentProps | Readonly<OverviewComponentProps>) {
        super(props);
    }
    
    public render() {
        const headerEntries = Object.entries(clusterDetailHeadings)
        const filteredHeaders: Array<[string, any]> = []
        if (this.props.item.length != 0) {
            headerEntries.filter((element: [string, any]) => {
                if (this.props.item[element[0]] != null &&
                    this.props.item[element[0]].toString().length > 0) {
                        filteredHeaders.push(element)
                    }
            })
            return (
                <Stack styles={contentStackStylesNormal}>
                    <Text variant="xxLarge">{this.props.clusterName}</Text>
                    <Stack horizontal>
                        <Stack id="Headers" styles={KeyColumnStyle}>
                        {filteredHeaders.map((value: [string, any], index: number) => (
                            <ClusterDetailCell style={KeyStyle} key={index} value={value[1]} />
                            )
                        )}
                        </Stack>
                        
                        <Stack id="Colons" styles={KeyColumnStyle}>
                        {Array(filteredHeaders.length).fill(':').map((value: [string], index: number) => (
                            <ClusterDetailCell style={KeyStyle} key={index} value={value} />
                            )
                        )}
                        </Stack>
                        
                        <Stack id="Values" styles={ValueColumnStyle}>
                        {filteredHeaders.map((value: [string, any], index: number) => (
                            <ClusterDetailCell style={ValueStyle}
                            key={index}
                            value={this.props.item[value[0]]} />
                            )
                        )}
                        </Stack>
                    </Stack>
                </Stack>
            );
        } else {
            return (
                <Stack>
                <Shimmer styles={headShimmerStyle} shimmerElements={headerShimmer} width="25%"></Shimmer>
                {headerEntries.map(header => (
                    <Shimmer key={header[0]} styles={ShimmerStyle} shimmerElements={rowShimmer} width="75%"></Shimmer>
                    )
                )}
                </Stack>
                )
            }
        }
    }