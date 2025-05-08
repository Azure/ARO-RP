import { useMemo } from "react"
import { Stack, ShimmeredDetailsList, SelectionMode } from "@fluentui/react"
import { contentStackStylesNormal } from "../App"
import { IClusterDetails } from "../ClusterDetailList"

interface OverviewComponentProps {
  item: any
  clusterName: string
}

const clusterDetailHeadings: IClusterDetails = {
  apiServerVisibility: "ApiServer Visibility",
  apiServerURL: "ApiServer URL",
  architectureVersion: "Architecture Version",
  consoleLink: "Console Link",
  createdAt: "Created At",
  createdBy: "Created By",
  failedProvisioningState: "Failed Provisioning State",
  infraId: "Infra Id",
  lastAdminUpdateError: "Last Admin Update Error",
  lastModifiedAt: "Last Modified At",
  lastModifiedBy: "Last Modified By",
  lastProvisioningState: "Last Provisioning State",
  location: "Location",
  name: "Name",
  provisioningState: "Provisioning State",
  resourceId: "Resource Id",
  version: "Version",
  installStatus: "Installation Status",
}

interface ClusterDetailItem {
  key: number
  name: string
  value: string
}

export function OverviewComponent(props: OverviewComponentProps) {
  const items: ClusterDetailItem[] = useMemo(
    () =>
      Object.entries(clusterDetailHeadings).map(([property, heading], index) => ({
        key: index,
        name: heading,
        value: props.item[property],
      })),
    [props.item]
  )

  return (
    <Stack styles={contentStackStylesNormal}>
      <Stack horizontal>
        <ShimmeredDetailsList
          compact={true}
          className="clusterOverviewList"
          items={items}
          columns={[
            { key: "name", name: "Name", fieldName: "name", minWidth: 200, isRowHeader: true },
            {
              key: "value",
              name: "Value",
              fieldName: "value",
              minWidth: 300,
              onRender: (item) => <span>{item.value || "-"}</span>,
            },
          ]}
          enableShimmer={props.item.length == 0}
          selectionMode={SelectionMode.none}
          isHeaderVisible={false}
        />
      </Stack>
    </Stack>
  )
}
