import React from "react";
import { Box, Stack, Flex, useColorModeValue } from "@chakra-ui/core";
import { brandColors } from "src/theme/theme";
import { useLocale } from "src/hooks";
import { LOCALE } from "src/types";
import LanguageSwitcher from "../LanguageSwitcher";
import ColorModeSwitcher from "../ColorModeSwitcher";

const Footer = () => {
  const [locale, setLocale] = useLocale();

  const color = useColorModeValue(
    brandColors.light,
    brandColors.dark
  );

  const onLocaleChange = (locale: LOCALE) => setLocale(locale);

  return (
    <Flex
      as="footer"
      align="center"
      justify="space-between"
      wrap="wrap"
      padding="1.5rem"
      borderTopWidth="1px"
      borderColor={color.primary}
    >
      <Box display="block" mt={{ base: 4, md: 0 }}>
        <Stack spacing={2} align="center" isInline>
          <ColorModeSwitcher flex="1" />
          <LanguageSwitcher
            flex="1"
            locale={locale}
            onLocaleChange={onLocaleChange}
          />
        </Stack>
      </Box>
    </Flex>
  );
};

export default Footer;
