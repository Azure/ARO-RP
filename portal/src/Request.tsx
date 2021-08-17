import axios, {AxiosResponse} from "axios"

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
  } catch (e) {
    let err = e.response as AxiosResponse
    return OnError(err)
  }
}

export const FetchClusterInfo = async (subscription: string, resourceGroup: string, name: string): Promise<AxiosResponse | null> => {
  try {
    const result = await axios("/api/" + subscription + "/" + resourceGroup +  "/" + name + "/clusterinfo")
    return result
  } catch (e) {
    let err = e.response as AxiosResponse
    return OnError(err)
  }
}

export const FetchInfo = async (): Promise<AxiosResponse | null> => {
  try {
    const result = await axios("/api/info")
    return result
  } catch (e) {
    let err = e.response as AxiosResponse
    return OnError(err)
  }
}

export const ProcessLogOut = async (): Promise<any> => {
  try {
    const result = await axios({method: "POST", url: "/api/logout"})
    return result
  } catch (e) {
    let err = e.response as AxiosResponse
    console.log(err)
  }
  document.location.href = "/api/login"
}

export const RequestKubeconfig = async (
  csrfToken: string,
  clusterID: string
): Promise<AxiosResponse | null> => {
  try {
    const result = await axios({
      method: "POST",
      url: clusterID + "/kubeconfig/new",
      headers: {
        "X-CSRF-Token": csrfToken,
      },
    })
    return result
  } catch (e) {
    let err = e.response as AxiosResponse
    return OnError(err)
  }
}
