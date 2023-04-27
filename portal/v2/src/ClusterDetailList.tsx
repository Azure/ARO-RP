import { Component } from "react"
import React from "react"
import { OverviewWrapper } from "./ClusterDetailListComponents/OverviewWrapper"
import { NodesWrapper } from "./ClusterDetailListComponents/NodesWrapper"
import { MachinesWrapper } from "./ClusterDetailListComponents/MachinesWrapper"
import { MachineSetsWrapper } from "./ClusterDetailListComponents/MachineSetsWrapper"
import { Statistics } from "./ClusterDetailListComponents/Statistics/Statistics"
import { ClusterOperatorsWrapper } from "./ClusterDetailListComponents/ClusterOperatorsWrapper";

import { ICluster } from "./App"

interface ClusterDetailComponentProps {
  item: IClusterDetails
  cluster: ICluster | null
  isDataLoaded: boolean
  detailPanelVisible: string
}

export interface IClusterDetails {
  apiServerVisibility: string
  apiServerURL: string
  architectureVersion: string
  consoleLink: string
  createdAt: string
  createdBy: string
  failedProvisioningState: string
  infraId: string
  lastAdminUpdateError: string
  lastModifiedAt: string
  lastModifiedBy: string
  lastProvisioningState: string
  location: string
  name: string
  resourceId: string
  provisioningState: string
  version: string
  installStatus: string
}

export interface WrapperProps {
  currentCluster: ICluster | null
  detailPanelSelected: string
  loaded: boolean
}

interface IClusterDetailComponentState {
  item: IClusterDetails // why both state and props?
  detailPanelSelected: string
}

const detailComponents: Map<string, any> = new Map<string, any>([
    ["nodes", NodesWrapper],
    ["machines", MachinesWrapper],
    ["machinesets", MachineSetsWrapper],
    ["clusteroperators", ClusterOperatorsWrapper],
    ["statistics", Statistics]
])

export class ClusterDetailComponent extends Component<ClusterDetailComponentProps, IClusterDetailComponentState> {

  constructor(props: ClusterDetailComponentProps | Readonly<ClusterDetailComponentProps>) {
    super(props)
  }

  public render() {
    if (this.props.cluster != undefined && this.props.detailPanelVisible != undefined) {
      const panel = this.props.detailPanelVisible.toLowerCase()
      if (panel == "overview") {
        return (
          <OverviewWrapper
            clusterName={this.props.cluster.name}
            currentCluster={this.props.cluster!}
            detailPanelSelected={panel}
            loaded={this.props.isDataLoaded}
          />
        )
      } else if (panel.includes("statistics")) {
        const StatisticsView = detailComponents.get("statistics")
        const type = panel.substring(0,panel.indexOf("statistics"))
        return (
          <StatisticsView currentCluster={this.props.cluster!} detailPanelSelected={panel} loaded = {this.props.isDataLoaded} statisticsType = {type}/>
        )
      } else {
        const DetailView = detailComponents.get(panel)
        return (
          <DetailView currentCluster={this.props.cluster!} detailPanelSelected={panel} loaded={this.props.isDataLoaded}/>
        )
      }
    }
  }
}

export const MemoisedClusterDetailListComponent = React.memo(ClusterDetailComponent)
