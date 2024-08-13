import { useId, useBoolean } from "@fluentui/react-hooks"
import {
  Popup,
  Layer,
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
import React, {
  useState,
  useImperativeHandle,
  useEffect,
  forwardRef,
  MutableRefObject,
} from "react"
import { RequestSSH } from "./Request"

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
  root: {
    background: "white",
    left: "50%",
    maxWidth: "500px",
    position: "absolute",
    top: "50%",
    transform: "translate(-50%, -50%)",
    border: "1px solid #CCC",
    boxShadow:
      "rgba(0, 0, 0, 0.22) 0px 25.6px 57.6px 0px, rgba(0, 0, 0, 0.18) 0px 4.8px 14.4px 0px",
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
  const [isPopupVisible, { setTrue: showPopup, setFalse: hidePopup }] = useBoolean(false)
  const titleId = useId("title")
  const [update, { setTrue: requestSSH, setFalse: sshRequested }] = useBoolean(false)
  const [resourceID, setResourceID] = useState("")
  const [machineName, setMachineName] = useState<IDropdownOption>()
  const [requestable, { setTrue: setRequestable, setFalse: setUnrequestable }] = useBoolean(false)
  const [data, setData] = useState<{ command: string; password: string } | null>()
  const [error, setError] = useState<Response | null>(null)

  useImperativeHandle(ref, () => ({
    LoadSSH: (item: string) => {
      setUnrequestable()
      setData(null)
      setError(null)
      showPopup()
      setResourceID(item)
    },
    hidePopup,
  }))

  useEffect(() => {
    async function fetchData() {
      try {
        setError(null)
        const result = await RequestSSH(csrfToken.current, machineName?.key as string, resourceID)
        setData(await result.json())
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
        <Dropdown
          id="sshDropdown"
          label={`Machine Selection`}
          onChange={onChange}
          options={machineOptions}
        />
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
      {isPopupVisible && (
        <Layer>
          <Popup
            className={contentStyles.root}
            role="dialog"
            aria-modal="true"
            onDismiss={hidePopup}
            enableAriaHiddenSiblings={true}>
            <div className={contentStyles.header} id="sshModal">
              <span id={titleId}>SSH Access</span>
              <IconButton
                styles={iconButtonStyles}
                iconProps={cancelIcon}
                ariaLabel="Close popup modal"
                onClick={hidePopup}
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
          </Popup>
        </Layer>
      )}
    </div>
  )
})

SSHModal.displayName = "sshmodal"
