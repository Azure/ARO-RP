import axios, { AxiosResponse } from "axios"
import { ICluster } from "./App"

const OnError = (err: AxiosResponse): AxiosResponse | null => {
  if (err.status === 403) {
    document.location.href = "/api/login"
    return null
  } else {
    return err
  }
}

export const FetchClusters = async (): Promise<AxiosResponse | null> => {
  try {
    const result = await axios("/api/clusters")
    return result
  } catch (e: any) {
    const err = e.response as AxiosResponse
    return OnError(err)
  }
}

export const FetchClusterInfo = async (cluster: ICluster): Promise<AxiosResponse | null> => {
  try {
    const result = await axios(
      "/api/" + cluster.subscription + "/" + cluster.resourceGroup + "/" + cluster.name
    )
    return result
  } catch (e: any) {
    const err = e.response as AxiosResponse
    return OnError(err)
  }
}

export const FetchInfo = async (): Promise<AxiosResponse | null> => {
  try {
    const result = await axios("/api/info")
    return result
  } catch (e: any) {
    const err = e.response as AxiosResponse
    return OnError(err)
  }
}

export const FetchNodes = async (cluster: ICluster): Promise<AxiosResponse | null> => {
  try {
    const result = await axios(
      "/api/" + cluster.subscription + "/" + cluster.resourceGroup + "/" + cluster.name + "/nodes")
    return result
  } catch (e: any) {
    const err = e.response as AxiosResponse
    return OnError(err)
  }
}

export const FetchMachines = async (cluster: ICluster): Promise<AxiosResponse | null> => {
  try {
    const result = await axios(
      "/api/" + cluster.subscription + "/" + cluster.resourceGroup + "/" + cluster.name + "/machines")
    return result
  } catch (e: any) {
    const err = e.response as AxiosResponse
    return OnError(err)
  }
}

export const FetchMachineSets = async (cluster: ICluster): Promise<AxiosResponse | null> => {
  try {
    const result = await axios(
      "/api/" + cluster.subscription + "/" + cluster.resourceGroup + "/" + cluster.name + "/machine-sets")
    return result
  } catch (e: any) {
    const err = e.response as AxiosResponse
    return OnError(err)
  }
}

export const FetchRegions = async (): Promise<AxiosResponse | null> => {
  try {
    const result = await axios("/api/regions")
    return result
  } catch (e: any) {
    let err = e.response as AxiosResponse
    return OnError(err)
  }
}

export const ProcessLogOut = async (): Promise<any> => {
  try {
    const result = await axios({method: "POST", url: "/api/logout"})
    return result
  } catch (e: any) {
    const err = e.response as AxiosResponse
    console.log(err)
  }
  document.location.href = "/api/login"
}

export const RequestKubeconfig = async (
  csrfToken: string,
  resourceID: string
): Promise<AxiosResponse | null> => {
  try {
    const result = await axios({
      method: "POST",
      url: resourceID + "/kubeconfig/new",
      headers: {
        "X-CSRF-Token": csrfToken,
      },
    })
    return result
  } catch (e: any) {
    const err = e.response as AxiosResponse
    return OnError(err)
  }
}
