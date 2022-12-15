import { TimePicker, IComboBox, DatePicker, IDatePickerStyles, Stack, Text, TooltipHost, IStackStyles, IconButton, mergeStyleSets } from "@fluentui/react"
import { iconButtonStyles } from "./Statistics"

export function GraphOptionsComponent(props: {duration: string, setDuration: React.Dispatch<React.SetStateAction<string>> , endDate: Date, setEndDate: React.Dispatch<React.SetStateAction<Date>>}): any {    
    const calculatorAddition =  { iconName: "CalculatorAddition" }
    const calculatorSubtract =  { iconName: "CalculatorSubtract" }
    const _increaseDuration = () => {
      switch(props.duration) {
        case "1m":
          props.setDuration("5m")
          break
        case "5m":
          props.setDuration("10m")
          break
        case "10m":
          props.setDuration("30m")
          break
        case "30m":
          props.setDuration("1h")
          break
        case "1h":
          props.setDuration("2h")
          break
        case "2h":
          props.setDuration("6h")
          break
        case "6h":
          props.setDuration("12h")
          break
        case "12h":
          props.setDuration("1d")
          break
        case "1d":
          props.setDuration("2d")
          break
        case "2d":
          props.setDuration("1w")
          break
        case "1w":
          props.setDuration("2w")
          break
        case "2w":
          props.setDuration("4w")
          break
        case "4w":
          props.setDuration("8w")
          break
      }
    }
    const _decreaseDuration = () => {
      switch(props.duration) {
        case "5m":
          props.setDuration("1m")
          break
        case "10m":
          props.setDuration("5m")
          break
        case "30m":
          props.setDuration("10m")
          break
        case "1h":
          props.setDuration("30m")
          break
        case "2h":
          props.setDuration("1h")
          break
        case "6h":
          props.setDuration("2h")
          break
        case "12h":
          props.setDuration("6h")
          break
        case "1d":
          props.setDuration("12h")
          break
        case "2d":
          props.setDuration("1d")
          break
        case "1w":
          props.setDuration("2d")
          break
        case "2w":
          props.setDuration("1w")
          break
        case "4w":
          props.setDuration("2w")
          break
        case "8w":
          props.setDuration("4w")
          break
      } 
    }
    const durationStyle: Partial<IStackStyles> = {
      root: {
        alignSelf: "flex-start",
        border: "2px",
        marginLeft: "3px",
        marginRight: "3px",
      },
    }
    const dateTimePickerStyles = mergeStyleSets({
      iconContainer: {
        marginLeft: 5,
      },
    })
    const classNames = mergeStyleSets({
      iconContainer: {
        margin: "0px 0px",
        height: 25,
        width: "90px",
      },
    })
  
    const datePickerStyles: Partial<IDatePickerStyles> = { root: { maxWidth: 145, marginTop: 5, } };
    const timePickerStyles: Partial<IDatePickerStyles> = { root: { maxWidth: 75, marginLeft: 5, marginRight: 5 } };
    const onTimeChange = (event: React.FormEvent<IComboBox>, date: Date) => {        
      var localDate = new Date()
      localDate.setFullYear(props.endDate.getFullYear(), props.endDate.getMonth(), props.endDate.getDate())
      localDate.setHours(date.getHours());
      localDate.setMinutes(date.getMinutes());
      props.setEndDate(localDate)        
     };
    const onDateChange = (date: Date | null | undefined): void => { 
      let localDate: Date = new Date();
      localDate.setFullYear(date!.getFullYear(), date!.getMonth(), date!.getDate())
      localDate.setHours(props.endDate.getHours());
      localDate.setMinutes(props.endDate.getMinutes());
      props.setEndDate(localDate)  
     };
     
    return (
      <>
      <Stack horizontal verticalAlign="center">
      <Stack horizontal verticalAlign="center" className={classNames.iconContainer} /* style={{ boxShadow: theme.effects.elevation8 }}*/>
          <TooltipHost>
            <IconButton
              onClick={_decreaseDuration}
              iconProps={calculatorSubtract}
              styles={iconButtonStyles}
            />
          </TooltipHost>
          <TooltipHost>
            <Text styles={durationStyle}>{props.duration}</Text>
          </TooltipHost>
          <TooltipHost >
            <IconButton
              onClick={_increaseDuration}
              iconProps={calculatorAddition}
              styles={iconButtonStyles}
            />
          </TooltipHost>
        </Stack>
        <Stack horizontal verticalAlign="center" className={dateTimePickerStyles.iconContainer}  /* style={{ boxShadow: theme.effects.elevation8 }}*/>
          <DatePicker
            styles={datePickerStyles}
            placeholder="End Date"
            ariaLabel="End Date"
            onSelectDate={onDateChange}
            value={props.endDate}
            allowTextInput
          />
          <TimePicker
            styles={timePickerStyles}
            allowFreeform
            placeholder={timeToString(props.endDate)}
            autoComplete="on"
            onChange={onTimeChange}
            defaultValue={props.endDate}
            useComboBoxAsMenuWidth
          />
        </Stack>
      </Stack>
      </>
    )
  }

  function timeToString(date: Date): string {
    var str
    let hourString = date.getHours().toString()
    if (hourString.length === 1) {
      str = "0" + hourString + ":"
    } else {
      str = hourString + ":"
    }
    let minuteString = date.getMinutes().toString()
    if (minuteString.length === 1) {
      str = str + "0" + minuteString
    } else {
      str += minuteString
    }

    return str
  }

  export const convertToUTC = (date: Date): Date => {
    let localDate = new Date()
    localDate.setFullYear(date.getUTCFullYear(), date.getUTCMonth(), date.getUTCDate())
    localDate.setHours(date.getUTCHours())
    localDate.setMinutes(date.getUTCMinutes())
    
    return localDate
  }