import React from "react";
import {defineMessages, useIntl} from "react-intl";
import {IconButton, useColorMode, useColorModeValue} from "@chakra-ui/core";
import {FaMoon, FaSun} from "react-icons/fa";

const messages = defineMessages({
  toggle: {
    id: "app.color_mode_switcher.toggle",
    defaultMessage: "Switch to {mode} mode",
  },
  dark: {
    id: "app.color_mode_switcher.mode_dark",
    defaultMessage: "Dark",
  },
  light: {
    id: "app.color_mode_switcher.mode_light",
    defaultMessage: "Light",
  },
});

export enum MODE {
  dark = "dark",
  light = "light",
}

type Omit<T, K> = Pick<T, Exclude<keyof T, K>>;
type Props = React.ComponentProps<typeof IconButton>;

function ColorModeSwitcher(props: Omit<Props, "aria-label">) {
  const intl = useIntl();
  const {toggleColorMode} = useColorMode();

  const colorMode = useColorModeValue(
    intl.formatMessage(messages.dark),
    intl.formatMessage(messages.light)
  );
  const SwitchIcon = useColorModeValue(FaMoon, FaSun);

  const label = intl.formatMessage(messages.toggle, {mode: colorMode});

  return (
    <IconButton
      aria-label={label as string}
      size="md"
      fontSize="lg"
      variant="ghost"
      marginLeft="2"
      onClick={toggleColorMode}
      icon={<SwitchIcon/>}
      {...props}
    />
  );
}

export default ColorModeSwitcher;
