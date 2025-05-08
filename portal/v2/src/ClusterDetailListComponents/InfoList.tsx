import {
  IShimmerStyles,
  IStackItemStyles,
  IStackStyles,
  ShimmerElementType,
  Stack,
  Text,
} from "@fluentui/react"
import { contentStackStylesNormal } from "../App"

export const ShimmerStyle: Partial<IShimmerStyles> = {
  root: {
    margin: "11px 0",
  },
}

export const headShimmerStyle: Partial<IShimmerStyles> = {
  root: {
    margin: "15px 0",
  },
}

export const headerShimmer = [{ type: ShimmerElementType.line, height: 32, width: "25%" }]

export const rowShimmer = [{ type: ShimmerElementType.line, height: 18, width: "75%" }]

export const KeyColumnStyle: Partial<IStackStyles> = {
  root: {
    paddingTop: 10,
    paddingRight: 15,
  },
}

export const ValueColumnStyle: Partial<IStackStyles> = {
  root: {
    paddingTop: 10,
  },
}

export const KeyStyle: IStackItemStyles = {
  root: {
    fontStyle: "bold",
    alignSelf: "flex-start",
    fontVariantAlternates: "bold",
    color: "grey",
    paddingBottom: 10,
  },
}

export const ValueStyle: IStackItemStyles = {
  root: {
    paddingBottom: 10,
  },
}

function Column(value: any): any {
  if (typeof value.value == typeof " ") {
    return (
      <Stack.Item styles={value.style}>
        <Text styles={value.style} variant={"medium"}>
          {value.value}
        </Text>
      </Stack.Item>
    )
  }
}

export const InfoList = (props: { headers: any; object: any; title: string; titleSize: any }) => {
  const headerEntries = Object.entries(props.headers)
  const filteredHeaders: Array<[string, any]> = []
  headerEntries.filter((element: [string, any]) => {
    if (props.object[element[0]] != null && props.object[element[0]].toString().length > 0) {
      filteredHeaders.push(element)
    }
  })
  return (
    <Stack styles={contentStackStylesNormal}>
      <Text variant={props.titleSize}>{props.title}</Text>
      <Stack horizontal>
        <Stack styles={KeyColumnStyle}>
          {filteredHeaders.map((value: [string, any], index: number) => (
            <Column style={KeyStyle} key={index} value={value[1]} />
          ))}
        </Stack>

        <Stack styles={KeyColumnStyle}>
          {Array(filteredHeaders.length)
            .fill(":")
            .map((value: [string], index: number) => (
              <Column style={KeyStyle} key={index} value={value} />
            ))}
        </Stack>

        <Stack styles={ValueColumnStyle}>
          {filteredHeaders.map((value: [string, any], index: number) => (
            <Column style={ValueStyle} key={index} value={props.object[value[0]]} />
          ))}
        </Stack>
      </Stack>
    </Stack>
  )
}

export const MultiInfoList = (props: any) => {
  return props.items.map((item: { [key: string]: any }) => {
    return (
      <InfoList
        key={item.key}
        headers={props.headers}
        object={item}
        title={item[props.subProp]}
        titleSize={item[props.titleSize]}
      />
    )
  })
}
