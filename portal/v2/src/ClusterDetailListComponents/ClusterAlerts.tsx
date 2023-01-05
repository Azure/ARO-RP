import { IShimmerStyles, Shimmer, ShimmerElementType } from '@fluentui/react/lib/Shimmer';
import { Component } from "react"
import { Stack, Text, IStackStyles, IStackItemStyles } from '@fluentui/react';
import { contentStackStylesNormal } from "../App"
import { ClusterAlertsMap } from "../ClusterDetailList"

interface ClusterAlertsComponentProps {
    item: any
    clusterName: string
}

interface IClusterAlertsComponentState {
    item: typeof ClusterAlertsMap
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

const clusterAlertHeadings: typeof ClusterAlertsMap = {}


function ClusterDetailCell(
    value: any,
    ): any {
        if (typeof (value.value) == typeof (" ")) {
            return <Stack.Item id="ClusterDetailCell" styles={value.style}>
            <Text styles={value.style} variant={'medium'}>{value.value}</Text>
            </Stack.Item>
        }
    }

function showClusterAlerts(alertsData: any) {
        let entries = alertsData
        return (
        <Stack styles={contentStackStylesNormal}>
          <Stack horizontal>
            <Stack styles={KeyColumnStyle}>
              {entries.map((value: any, index: number) => (
                <ClusterDetailCell style={KeyStyle} key={index} value={value.alertname} />
              ))}
            </Stack>

            <Stack styles={KeyColumnStyle}>
              {Array(entries.length)
                .fill(":")
                .map((value: any, index: number) => (
                  <ClusterDetailCell style={KeyStyle} key={index} value={value} />
                ))}
            </Stack>

            <Stack styles={ValueColumnStyle}>
              {entries.map((value: any, index: number) => {
                 return (
                     <ClusterDetailCell
                        style={ValueStyle}
                        key={index}
                        value={
                          value.status != null &&
                          value.status.length > 0
                            ? value.status
                            : "Undefined"
                        }
                      />
                    )
              }
            )}
            </Stack>
          </Stack>
        </Stack>
      )
      }

function clusterAlertsShimmer() {
        const headerEntries = Object.entries(clusterAlertHeadings)
      return (
        <Stack>
          <Shimmer
            styles={headShimmerStyle}
            shimmerElements={headerShimmer}
            width="25%"></Shimmer>
          {headerEntries.map((value: [any, any], index: number) => (
            <Shimmer
              styles={ShimmerStyle}
              key={index}
              shimmerElements={rowShimmer}
              width="75%"></Shimmer>
          ))}
        </Stack>
      )
      }

export class ClusterAlertsComponent extends Component<ClusterAlertsComponentProps, IClusterAlertsComponentState > {

    constructor(props: ClusterAlertsComponentProps | Readonly<ClusterAlertsComponentProps>) {
        super(props);
    }
    public render() {
        {
            if (this.props.item.length != 0) {
              return showClusterAlerts(this.props.item)
            } else  {
              return clusterAlertsShimmer()
            }
         }
    }
}