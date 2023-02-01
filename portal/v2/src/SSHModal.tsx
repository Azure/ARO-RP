import { useId, useBoolean } from "@fluentui/react-hooks"
import {
  Modal,
  getTheme,
  mergeStyleSets,
  FontWeights,
  IIconProps,
  Dropdown,
  IDropdownOption,
  TextField,
  MessageBar,
  MessageBarType,
  Stack,
  FontSizes,
} from "@fluentui/react"
import { PrimaryButton, IconButton } from "@fluentui/react/lib/Button"
import axios, { AxiosResponse } from "axios"
import React, {
  useState,
  useImperativeHandle,
  useEffect,
  forwardRef,
  MutableRefObject,
} from "react"

const cancelIcon: IIconProps = { iconName: "Cancel" }
const copyIcon: IIconProps = { iconName: "Copy" }

const machineOptions = [
  { key: 0, text: "master-0" },
  { key: 1, text: "master-1" },
  { key: 2, text: "master-2" },
]

type SSHModalProps = {
  csrfToken: MutableRefObject<string>
}

const theme = getTheme()
const contentStyles = mergeStyleSets({
  container: {
    display: "flex",
    flexFlow: "column nowrap",
    alignItems: "stretch",
  },
  header: [
    {
      flex: "1 1 auto",
      borderTop: `4px solid ${theme.palette.themePrimary}`,
      color: theme.palette.neutralPrimary,
      display: "flex",
      alignItems: "center",
      fontSize: FontSizes.xLargePlus,
      fontWeight: FontWeights.semibold,
      padding: "12px 12px 14px 24px",
    },
  ],
  body: {
    flex: "4 4 auto",
    padding: "0 24px 24px 24px",
    overflowY: "hidden",
    selectors: {
      "p": { margin: "14px 0" },
      "p:first-child": { marginTop: 0 },
      "p:last-child": { marginBottom: 0 },
    },
  },
})

const iconButtonStyles = {
  root: {
    color: theme.palette.neutralPrimary,
    marginLeft: "auto",
    marginTop: "4px",
    marginRight: "2px",
  },
  rootHovered: {
    color: theme.palette.neutralDark,
  },
}

const sshDocs: string =
  "https://msazure.visualstudio.com/AzureRedHatOpenShift/_wiki/wikis/ARO.wiki/136823/ARO-SRE-portal?anchor=ssh-(elevated)"

export const SSHModal = forwardRef<any, SSHModalProps>(({ csrfToken }, ref) => {
  const [isModalOpen, { setTrue: showModal, setFalse: hideModal }] = useBoolean(false)

  const titleId = useId("title")
  const [update, { setTrue: requestSSH, setFalse: sshRequested }] = useBoolean(false)
  const [resourceID, setResourceID] = useState("")
  const [machineName, setMachineName] = useState<IDropdownOption>()
  const [requestable, { setTrue: setRequestable, setFalse: setUnrequestable }] = useBoolean(false)
  const [data, setData] = useState<{ command: string; password: string } | null>()
  const [error, setError] = useState<AxiosResponse | null>(null)

  useImperativeHandle(ref, () => ({
    LoadSSH: (item: string) => {
      setUnrequestable()
      setData(null)
      setError(null)
      showModal()
      setResourceID(item)
    },
  }))

  useEffect(() => {
    async function fetchData() {
      try {
        setError(null)
        const result = await axios({
          method: "post",
          url: resourceID + "/ssh/new",
          data: {
            master: machineName?.key,
          },
          headers: { "X-CSRF-Token": csrfToken.current },
        })
        setData(result.data)
        setRequestable()
      } catch (error: any) {
        setRequestable()
        setError(error.response)
      }
    }
    if (update && machineName) {
      sshRequested()
      fetchData()
    }
    return
  }, [resourceID, machineName, csrfToken, update, sshRequested, setRequestable])

  const onChange = (
    event: React.FormEvent<HTMLDivElement>,
    option?: IDropdownOption<any>
  ): void => {
    setMachineName(option)
    setRequestable()
  }

  const errorBar = (): any => {
    return (
      <MessageBar
        messageBarType={MessageBarType.error}
        isMultiline={false}
        onDismiss={() => setError(null)}
        dismissButtonAriaLabel="Close">
        {error?.statusText}
      </MessageBar>
    )
  }

  const selectionField = (): any => {
    return (
      <Stack tokens={{ childrenGap: 15 }}>
        <Dropdown id ="sshDropdown" label={`Machine Selection`} onChange={onChange} options={machineOptions} />
        <PrimaryButton onClick={requestSSH} id="sshButton" text="Request" disabled={!requestable} />
      </Stack>
    )
  }

  const dataResult = (): any => {
    return (
      <div>
        <Stack id="sshCommand">
          <Stack horizontal verticalAlign={"end"}>
            <Stack.Item grow>
              <TextField label="Command" value={data?.command} readOnly />
            </Stack.Item>
            <Stack.Item>
              <IconButton
                iconProps={copyIcon}
                ariaLabel="Copy command"
                onClick={() => {
                  if (data) {
                    navigator.clipboard.writeText(data.command)
                  }
                }}
              />
            </Stack.Item>
          </Stack>
          <Stack horizontal verticalAlign={"end"}>
            <Stack.Item grow>
              <TextField
                label="Password"
                value={data?.password}
                type="password"
                canRevealPassword
                readOnly
              />{" "}
            </Stack.Item>
            <Stack.Item>
              <IconButton
                iconProps={copyIcon}
                ariaLabel="Copy password"
                onClick={() => {
                  if (data) {
                    navigator.clipboard.writeText(data.password)
                  }
                }}
              />
            </Stack.Item>
          </Stack>
        </Stack>
      </div>
    )
  }

  return (
    <div>
      <Modal
        titleAriaId={titleId}
        isOpen={isModalOpen}
        onDismiss={hideModal}
        isModeless={true}
        containerClassName={contentStyles.container}>
        <div className={contentStyles.header} id="sshModal">
          <span id={titleId}>SSH Access</span>
          <IconButton
            styles={iconButtonStyles}
            iconProps={cancelIcon}
            ariaLabel="Close popup modal"
            onClick={hideModal}
          />
        </div>

        <div className={contentStyles.body}>
          <p>
            Before requesting SSH access, please ensure you have read the{" "}
            <a href={sshDocs}>SSH docs</a>.
          </p>
          {error && errorBar()}
          {data ? dataResult() : selectionField()}
        </div>
      </Modal>
    </div>
  )
})

SSHModal.displayName = "sshmodal"
