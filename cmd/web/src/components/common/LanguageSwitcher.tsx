import React from "react";
import {Select} from "@chakra-ui/core";
import {LOCALE} from "src/types";

type SelectProps = React.ComponentProps<typeof Select>;

export interface LanguageProps {
  label: string;
  symbol: string;
}

const Language = (props: LanguageProps) => (
  <option
    className="emoji"
    role="img"
    aria-label={props.label ? props.label : ""}
    aria-hidden={props.label ? "false" : "true"}
    value={props.label}
  >
    {props.symbol}
  </option>
);

export interface Props {
  locale: LOCALE;
  onLocaleChange: (locale: LOCALE) => void;
}

function LanguageSwitcher({
                            locale,
                            onLocaleChange,
                            ...props
                          }: Props & SelectProps) {
  return (
    <Select
      bg="transparent"
      size="lg"
      variant="unstyled"
      onChange={(e) => onLocaleChange(e.target.value as LOCALE)}
      value={locale}
      {...props}
    >
      <Language label={LOCALE.en} symbol="🇺🇸"/>
      <Language label={LOCALE.pl} symbol="🇵🇱"/>
    </Select>
  );
}

export default LanguageSwitcher;
