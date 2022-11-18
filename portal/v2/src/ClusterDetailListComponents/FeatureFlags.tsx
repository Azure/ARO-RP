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
    item: FeatureFlags
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

const featureFlagsHeadings: FeatureFlags = {
  "aro.alertwebhook.enabled": "ARO Alertwebhook Enabled",
  "aro.autosizednodes.enable": "ARO Autosizednodes Enable",
  "aro.azuresubnets.enabled": "ARO Azuresubnets Enabled",
  "aro.azuresubnets.nsg.managed": "ARO Azuresubnets NSG Managed",
  "aro.azuresubnets.serviceendpoint.managed": "ARO Azuresubnets Serviceendpoint Managed",
  "aro.banner.enabled": "ARO Banner Enabled",
  "aro.checker.enabled": "ARO Checker Enabled",
  "aro.dnsmasq.enabled": "ARO Dnsmasq Enabled",
  "aro.genevalogging.enabled": "ARO Genevalogging Enabled",
  "aro.imageconfig.enabled": "ARO Imageconfig Enabled",
  "aro.machine.enabled": "ARO Machine Enabled",
  "aro.machinehealthcheck.enabled": "ARO Machinehealthcheck Enabled",
  "aro.machinehealthcheck.managed": "ARO Machinehealthcheck Managed",
  "aro.machineset.enabled": "ARO Machineset Enabled",
  "aro.monitoring.enabled": "ARO Monitoring Enabled",
  "aro.nodedrainer.enabled": "ARO Nodedrainer Enabled",
  "aro.pullsecret.enabled": "ARO Pullsecret Enabled",
  "aro.pullsecret.managed": "ARO Pullsecret Managed",
  "aro.rbac.enabled": "ARO RBAC Enabled",
  "aro.routefix.enabled": "ARO Routefix Enabled",
  "aro.storageaccounts.enabled": "ARO Storageaccounts Enabled",
  "aro.workaround.enabled": "ARO Workaround Enabled",
  "rh.srep.muo.enabled": "RH SREP MUO Enabled",
  "rh.srep.muo.managed": "RH SREP MUO Managed",
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
    
function opFeatureFlagsHeadings(item: any) {
        const headerEntries = Object.entries(featureFlagsHeadings)
        return (
        <Stack styles={contentStackStylesNormal}>
          <Stack horizontal>
            <Stack styles={KeyColumnStyle}>
              {headerEntries.map((value: any, index: number) => (
                <ClusterDetailCell style={KeyStyle} key={index} value={value[1]} />
              ))}
            </Stack>
      
            <Stack styles={KeyColumnStyle}>
              {Array(headerEntries.length)
                .fill(":")
                .map((value: any, index: number) => (
                  <ClusterDetailCell style={KeyStyle} key={index} value={value} />
                ))}
            </Stack>
      
            <Stack styles={ValueColumnStyle}>
              {headerEntries.map((value: [any, any], index: number) => {
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
      
function opFeatureFlagsShimmer() {
        const headerEntries = Object.entries(featureFlagsHeadings)
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
    
export class FeatureFlagsComponent extends Component<FeatureFlagsComponentProps, IFeatureFlagsComponentState> {
    
    constructor(props: FeatureFlagsComponentProps | Readonly<FeatureFlagsComponentProps>) {
        super(props);
    }
    
    public render() {
        {
            if (this.props.item.length != 0) {
              return opFeatureFlagsHeadings(this.props.item)
            } else  {
              return opFeatureFlagsShimmer()
            }
         }
    }
}