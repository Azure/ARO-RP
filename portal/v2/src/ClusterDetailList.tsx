import { Component } from "react"
import { OverviewWrapper } from './ClusterDetailListComponents/OverviewWrapper';
import { NodesWrapper } from './ClusterDetailListComponents/NodesWrapper';
import { MachinesWrapper } from "./ClusterDetailListComponents/MachinesWrapper";
import { MachineSetsWrapper } from "./ClusterDetailListComponents/MachineSetsWrapper";
import { ICluster } from "./App"

interface ClusterDetailComponentProps {
  item: any
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

interface IClusterDetailComponentState {
  item: IClusterDetails // why both state and props?
  detailPanelSelected: string
}

export class ClusterDetailComponent extends Component<ClusterDetailComponentProps, IClusterDetailComponentState> {

  constructor(props: ClusterDetailComponentProps | Readonly<ClusterDetailComponentProps>) {
    super(props);
  }

  public render() {
    switch (this.props.detailPanelVisible.toLowerCase()) {
      case "overview":
      {
        return (
          <OverviewWrapper clusterName= {this.props.item.name} currentCluster={this.props.cluster!} detailPanelSelected={this.props.detailPanelVisible} loaded={this.props.isDataLoaded}/>
        )
      }
      case "nodes":
        {
          return (
            <NodesWrapper currentCluster={this.props.cluster!} detailPanelSelected={this.props.detailPanelVisible} loaded={this.props.isDataLoaded}/>
          );
        }
        case "machines":
        {
          return (
            <MachinesWrapper currentCluster={this.props.cluster!} detailPanelSelected={this.props.detailPanelVisible} loaded={this.props.isDataLoaded}/>
          );
        }
        case "machinesets":
        {
          return (
            <MachineSetsWrapper currentCluster={this.props.cluster!} detailPanelSelected={this.props.detailPanelVisible} loaded={this.props.isDataLoaded}/>
          );
        }
      }
  }
}
