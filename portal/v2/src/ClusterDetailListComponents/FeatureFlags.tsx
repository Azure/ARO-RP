import { IShimmerStyles, Shimmer, ShimmerElementType } from '@fluentui/react/lib/Shimmer';
import { Component } from "react"
import { Stack, Text, IStackStyles, IStackItemStyles } from '@fluentui/react';
import { contentStackStylesNormal } from "../App"
import { FeatureFlags } from "../ClusterDetailList"

interface FeatureFlagsComponentProps {
    item: any
    clusterName: string
}

interface IFeatureFlagsComponentState {
    item: FeatureFlags[]
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

function ClusterDetailCell(
    value: any,
    ): any {
        if (typeof (value.value) == typeof (" ")) {
            return <Stack.Item id="ClusterDetailCell" styles={value.style}>
            <Text styles={value.style} variant={'medium'}>{value.value}</Text>
            </Stack.Item>
        }
    }

function showFeatureFlags(item: [FeatureFlags]) {
        const featureFlagsData = Object.entries(item)
        return (
        <Stack styles={contentStackStylesNormal}>
          <Stack horizontal>
            <Stack styles={KeyColumnStyle}>
              {featureFlagsData.map((value: any, index: number) => (
                <ClusterDetailCell style={KeyStyle} key={index} value={value[0]} />
              ))}
          </Stack>

            <Stack styles={KeyColumnStyle}>
              {Array(featureFlagsData.length)
                .fill(":")
                .map((value: any, index: number) => (
                  <ClusterDetailCell style={KeyStyle} key={index} value={value} />
                ))}
            </Stack>

            <Stack styles={ValueColumnStyle}>
              {featureFlagsData.map((value: [any, any], index: number) => {
                 return (
                     <ClusterDetailCell
                        style={ValueStyle}
                        key={index}
                        value={
                          item[value[0]] != null &&
                          item[value[0]].toString().length > 0
                            ? item[value[0]]
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

function showFeatureFlagsShimmer(length: number) {
      return (
        <Stack>
          <Shimmer
            styles={headShimmerStyle}
            shimmerElements={headerShimmer}
            width="25%"></Shimmer>
            {Array(length)
             .map((index: number) => (
            <Shimmer
              styles={ShimmerStyle}
              key={index}
              shimmerElements={rowShimmer}
              width="75%"></Shimmer>
          ))}
        </Stack>
      )}
export class FeatureFlagsComponent extends Component<FeatureFlagsComponentProps, IFeatureFlagsComponentState> {

    constructor(props: FeatureFlagsComponentProps | Readonly<FeatureFlagsComponentProps>) {
        super(props);
    }

    public render() {
        {
            if (this.props.item.length != 0) {
              return showFeatureFlags(this.props.item)
            } else  {
              return showFeatureFlagsShimmer(this.props.item.length)
            }
         }
    }
}