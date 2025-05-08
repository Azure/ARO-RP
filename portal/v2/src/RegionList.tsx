import { Component } from "react"
import {
  Nav,
  INavLinkGroup,
  INavStyles,
  IRenderGroupHeaderProps,
  IRenderFunction,
  Stack,
  Text,
  TextField,
} from "@fluentui/react"

const navStyles: Partial<INavStyles> = {
  chevronIcon: {
    visibility: "hidden",
  },
  navItem: {
    height: "35px",
  },
  link: {
    height: "35px",
  },
}

interface RegionComponentProps {
  item: IRegionList
}

interface IRegionComponentState {
  items: IRegion[]
}

export interface IRegionList {
  regions: IRegion[]
}

export interface IRegion {
  name: string
  url: string
}

export class RegionComponent extends Component<RegionComponentProps, IRegionComponentState> {
  private _allItems: IRegion[]

  constructor(props: RegionComponentProps | Readonly<RegionComponentProps>) {
    super(props)
    this._allItems = this.props.item.regions

    this.state = {
      items: this._allItems,
    }
  }

  public render() {
    const items = this.state.items
    var navGroups: INavLinkGroup[] = [
      {
        name: "Regions",
        links: items,
      },
    ]
    return (
      <Stack>
        <Nav
          onRenderGroupHeader={this._onRenderGroupHeader}
          styles={navStyles}
          groups={navGroups}
        />
        <Text>{this.props.item.regions.length == 0 && "No extra regions within this cloud"}</Text>
      </Stack>
    )
  }

  private _onChangeText = (
    ev: React.FormEvent<HTMLInputElement | HTMLTextAreaElement>,
    text?: string
  ): void => {
    this.setState({
      items: text
        ? this._allItems.filter((i) => i.name.toLowerCase().indexOf(text.toLowerCase()) > -1)
        : this._allItems,
    })
  }

  private _onRenderGroupHeader: IRenderFunction<IRenderGroupHeaderProps> = (props): JSX.Element => {
    return (
      <Stack>
        <h3>{props?.name}</h3>
        <TextField label="Filter by name:" onChange={this._onChangeText} />
      </Stack>
    )
  }
}
